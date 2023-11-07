package hangar

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/source"
	"github.com/cnrancher/hangar/pkg/types"
	"github.com/cnrancher/hangar/pkg/utils"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

type Syncer struct {
	*common

	au      *archive.Updater
	auMutex *sync.RWMutex
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

type SyncerOpts struct {
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

func NewSyncer(o *SyncerOpts) *Syncer {
	s := &Syncer{
		auMutex: &sync.RWMutex{},
		index:   archive.NewIndex(),

		SourceRegistry:    o.SourceProject,
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

func (s *Syncer) copy(ctx context.Context) {
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

	if err := s.updateIndex(); err != nil {
		logrus.Errorf("failed to write index file: %v", err)
	}
	s.au.Close()
}

func (s *Syncer) newSaveCacheDir() (string, error) {
	cd, err := os.MkdirTemp(archive.CacheDir(), "*")
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	logrus.Debugf("create save cache dir: %v", cd)
	return cd, nil
}

func (s *Syncer) recordFailedImage(name string) {
	s.common.failedImageListMutex.Lock()
	s.common.failedImageSet[name] = true
	s.common.failedImageListMutex.Unlock()
}

func (s *Syncer) updateIndex() error {
	s.au.SetIndex(s.index)
	return s.au.UpdateIndex()
}

// Run append images from registry server into local directory / hangar archive.
func (s *Syncer) Run(ctx context.Context) error {
	au, err := archive.NewUpdater(s.ArchiveName)
	if err != nil {
		return fmt.Errorf("failed to open archive %q: %w", s.ArchiveName, err)
	}
	s.au = au

	s.copy(ctx)
	if len(s.failedImageSet) != 0 {
		for i := range s.failedImageSet {
			fmt.Printf("%v\n", i)
		}
		return fmt.Errorf("some images failed to sync to %q", s.ArchiveName)
	}
	return nil
}

func (s *Syncer) worker(ctx context.Context, o any) {
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
		Infof("Syncing [%v]", obj.source.ReferenceNameWithoutTransport())
	err = obj.destination.Init(copyContext)
	if err != nil {
		return
	}
	err = obj.source.Copy(copyContext, obj.destination, s.common.imageSpecSet)
	if err != nil {
		return
	}

	// images copied to cache folder, write to archive file
	s.auMutex.Lock()
	defer s.auMutex.Unlock()
	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Debugf("Compressing [%v]", obj.destination.ReferenceNameWithoutTransport())
	err = s.au.Append(obj.destination.ReferenceNameWithoutTransport())
	if err != nil {
		return
	}
	s.index.Append(obj.source.GetCopiedImage())
	s.auMutex.Unlock()
	// delete cache dir
	err = os.RemoveAll(obj.destination.ReferenceNameWithoutTransport())
	if err != nil {
		err = fmt.Errorf(
			"failed to delete cache dir %q: %w",
			obj.destination.ReferenceNameWithoutTransport(), err)
	}

	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("Synced [%v] => [%v]",
			obj.source.ReferenceNameWithoutTransport(),
			s.ArchiveName)
}
