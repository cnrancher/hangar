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
	// AchiveFormat is the saved archive format
	ArchiveFormat archive.Format
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
	// AchiveFormat is the saved archive format
	ArchiveFormat archive.Format
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
		ArchiveFormat:     o.ArchiveFormat,
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
		object := &copyObject{
			id:     i,
			copier: nil,
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
	s.common.failedImageList = append(s.common.failedImageList, name)
	s.common.failedImageListMutex.Unlock()
}

func (s *Saver) writeIndex() error {
	return s.aw.WriteIndex(s.index)
}

// Run save images from registry server into local directory / tarball archive.
func (s *Saver) Run(ctx context.Context) error {
	aw, err := archive.NewWriter(s.ArchiveName, s.ArchiveFormat, archive.CreateTrunc)
	if err != nil {
		return fmt.Errorf("failed to create archive %q: %w", s.ArchiveName, err)
	}
	s.aw = aw

	s.copy(ctx)
	if len(s.failedImageList) != 0 {
		return fmt.Errorf("some images failed to save")
	}
	return nil
}

func (s *Saver) worker(ctx context.Context, id int) {
	defer s.common.waitGroup.Done()
	for {
		select {
		case <-ctx.Done():
			logrus.WithFields(logrus.Fields{"w": id}).
				Debugf("worker stopped: %v", ctx.Err())
			return
		case obj, ok := <-s.common.objectCh:
			if !ok {
				logrus.WithFields(logrus.Fields{"w": id}).
					Debugf("channel closed, release worker")
				return
			}
			if obj == nil {
				continue
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

			err = obj.source.Init(copyContext)
			if err != nil {
				s.common.errorCh <- err
				s.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
				cancel()
				continue
			}
			logrus.WithFields(logrus.Fields{"w": id}).
				Infof("Saving [%v]", obj.source.ReferenceName())
			err = obj.destination.Init(copyContext)
			if err != nil {
				s.common.errorCh <- err
				s.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
				cancel()
				continue
			}
			err = obj.source.Copy(copyContext, obj.destination, s.common.imageSpecSet)
			if err != nil {
				s.common.errorCh <- err
				s.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
				cancel()
				continue
			}

			// images copied to cache folder, write to archive file
			s.awMutex.Lock()
			logrus.WithFields(logrus.Fields{"w": id}).
				Debugf("Compressing [%v]", obj.destination.ReferenceNameWithoutTransport())
			err = s.aw.Write(obj.destination.ReferenceNameWithoutTransport())
			if err != nil {
				s.common.errorCh <- fmt.Errorf("failed to write cache dir to archive: %w", err)
				s.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
				cancel()
				s.awMutex.Unlock()
				continue
			}
			s.index.Append(obj.source.GetCopiedImage())
			s.awMutex.Unlock()
			// delete cache dir
			err = os.RemoveAll(obj.destination.ReferenceNameWithoutTransport())
			if err != nil {
				s.common.errorCh <- fmt.Errorf(
					"failed to delete cache dir %q: %w",
					obj.destination.ReferenceNameWithoutTransport(), err)
			}

			logrus.WithFields(logrus.Fields{"w": id}).
				Infof("Saved [%v] => [%v]",
					obj.source.ReferenceNameWithoutTransport(),
					s.ArchiveName)
			// Cancel the context after copy image
			cancel()
		}
	}
}
