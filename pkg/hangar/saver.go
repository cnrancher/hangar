package hangar

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/source"
	"github.com/cnrancher/hangar/pkg/types"
	"github.com/cnrancher/hangar/pkg/utils"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

// saveObject is the object for sending to worker pool when saving image
type saveObject struct {
	image       string
	source      *source.Source
	destination *destination.Destination
	timeout     time.Duration
	id          int
}

type Saver struct {
	*common

	aw      *archive.Writer
	awMutex *sync.RWMutex
	index   *archive.Index

	// Override the registry of source image to be copied
	SourceRegistry string
	// Override the project of source image to be copied
	SourceProject string
	// SharedBlobDirPath is the directory to save the shared blobs
	SharedBlobDirPath string
	// ArchiveName is the saved archive file name
	ArchiveName string
}

type SaverOpts struct {
	CommonOpts

	// Override the registry of source image to be copied
	SourceRegistry string
	// Override the project of source image to be copied
	SourceProject string
	// SharedBlobDirPath is the directory to save the shared blobs
	SharedBlobDirPath string
	// ArchiveName is the saved archive file name
	ArchiveName string
}

func NewSaver(o *SaverOpts) *Saver {
	s := &Saver{
		awMutex: &sync.RWMutex{},
		index:   archive.NewIndex(),

		SourceRegistry:    o.SourceRegistry,
		SourceProject:     o.SourceProject,
		SharedBlobDirPath: o.SharedBlobDirPath,
		ArchiveName:       o.ArchiveName,
	}
	if s.SharedBlobDirPath == "" {
		s.SharedBlobDirPath = archive.SharedBlobDir
	}
	s.common = newCommon(&o.CommonOpts)
	return s
}

func (s *Saver) copy(ctx context.Context) {
	s.common.initErrorHandler(ctx)
	s.common.initWorker(ctx, s.worker)
	for i, img := range s.common.images {
		object := &saveObject{
			id: i + 1,
		}
		sourceRegistry := utils.GetRegistryName(img)
		if s.SourceRegistry != "" {
			sourceRegistry = s.SourceRegistry
		}
		sourceProject := utils.GetProjectName(img)
		if s.SourceProject != "" {
			sourceProject = s.SourceProject
		}
		src, err := source.NewSource(&source.Option{
			Type:          types.TypeDocker,
			Registry:      sourceRegistry,
			Project:       sourceProject,
			Name:          utils.GetImageName(img),
			Tag:           utils.GetImageTag(img),
			SystemContext: &imagetypes.SystemContext{},
		})
		if err != nil {
			s.errorCh <- fmt.Errorf("failed to init source image: %w", err)
			s.recordFailedImage(img)
			continue
		}
		object.source = src

		cd, err := s.newSaveCacheDir()
		if err != nil {
			s.errorCh <- fmt.Errorf("failed to create cache dir: %w", err)
			s.recordFailedImage(img)
			continue
		}
		sd := path.Join(cd, s.SharedBlobDirPath)
		dest, err := destination.NewDestination(&destination.Option{
			Type:      types.TypeOci,
			Directory: cd,
			Name:      utils.GetImageName(img),
			Tag:       utils.GetImageTag(img),
			SystemContext: &imagetypes.SystemContext{
				OCISharedBlobDirPath: sd,
			},
		})
		if err != nil {
			s.errorCh <- fmt.Errorf("failed to init dest image: %w", err)
			s.recordFailedImage(img)
			continue
		}
		object.destination = dest
		s.common.objectCh <- object
	}

	close(s.common.objectCh)
	// Waiting for all images were copied
	s.common.waitGroup.Wait()
	close(s.common.errorCh)
	// Waiting for all error messages were handled properly
	s.common.errorWaitGroup.Wait()

	if err := s.writeIndex(); err != nil {
		logrus.Errorf("failed to write index file: %v", err)
	}
	s.aw.Close()
}

func (s *Saver) newSaveCacheDir() (string, error) {
	cd, err := os.MkdirTemp(archive.CacheDir(), "*")
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	logrus.Debugf("create save cache dir: %v", cd)
	return cd, nil
}

func (s *Saver) recordFailedImage(name string) {
	s.common.failedImageListMutex.Lock()
	s.common.failedImageSet[name] = true
	s.common.failedImageListMutex.Unlock()
}

func (s *Saver) writeIndex() error {
	return s.aw.WriteIndex(s.index)
}

// Run save images from registry server into local directory / hangar archive.
func (s *Saver) Run(ctx context.Context) error {
	aw, err := archive.NewWriter(s.ArchiveName)
	if err != nil {
		return fmt.Errorf("failed to create archive %q: %w", s.ArchiveName, err)
	}
	s.aw = aw

	s.copy(ctx)
	if len(s.failedImageSet) != 0 {
		for i := range s.failedImageSet {
			fmt.Printf("%v\n", i)
		}
		return fmt.Errorf("some images failed to save")
	}
	return nil
}

func (s *Saver) worker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*saveObject)
	if !ok {
		logrus.Errorf("skip object type(%T), data %v", o, o)
		return
	}

	var (
		copyContext context.Context
		cancel      context.CancelFunc
		err         error
	)
	if obj.timeout > 0 {
		copyContext, cancel = context.WithTimeout(ctx, obj.timeout)
	} else {
		copyContext, cancel = context.WithCancel(ctx)
	}
	defer func() {
		cancel()
		if err != nil {
			s.common.errorCh <- err
			s.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
		}
	}()

	err = obj.source.Init(copyContext)
	if err != nil {
		return
	}
	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("Saving [%v]", obj.source.ReferenceNameWithoutTransport())
	err = obj.destination.Init(copyContext)
	if err != nil {
		return
	}
	err = obj.source.Copy(copyContext, obj.destination, s.common.imageSpecSet)
	if err != nil {
		return
	}

	// images copied to cache folder, write to archive file
	s.awMutex.Lock()
	defer s.awMutex.Unlock()
	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Debugf("Compressing [%v]", obj.destination.ReferenceNameWithoutTransport())
	err = s.aw.Write(obj.destination.ReferenceNameWithoutTransport())
	if err != nil {
		return
	}
	s.index.Append(obj.source.GetCopiedImage())
	// delete cache dir
	err = os.RemoveAll(obj.destination.ReferenceNameWithoutTransport())
	if err != nil {
		err = fmt.Errorf(
			"failed to delete cache dir %q: %w",
			obj.destination.ReferenceNameWithoutTransport(), err)
	}

	logrus.WithFields(logrus.Fields{
		"IMG": obj.id,
	}).Infof("Saved [%v] => [%v]",
		obj.source.ReferenceNameWithoutTransport(),
		s.ArchiveName)
}
