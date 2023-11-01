package hangar

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/sirupsen/logrus"
)

type Loader struct {
	*common

	ar      *archive.Reader
	arMutex *sync.RWMutex
	index   *archive.Index

	// Specify the destination image registry.
	DestinationRegistry string
	// Specify the destination image project.
	DestinationProject string
	// Directory is the source archive directory
	Directory string
	// SharedBlobDirPath is the directory to save the shared blobs
	SharedBlobDirPath string
	// ArchiveName is the archive file name to be load
	ArchiveName string
}

type LoaderOpts struct {
	CommonOpts

	// Specify the destination image registry.
	DestinationRegistry string
	// Specify the destination image project.
	DestinationProject string
	// Directory is the source archive directory
	Directory string
	// SharedBlobDirPath is the directory to save the shared blobs
	SharedBlobDirPath string
	// ArchiveName is the archive file name to be load
	ArchiveName string
}

func NewLoader(o *LoaderOpts) *Loader {
	l := &Loader{
		DestinationRegistry: o.DestinationRegistry,
		DestinationProject:  o.DestinationProject,
		Directory:           o.Directory,
		SharedBlobDirPath:   o.SharedBlobDirPath,
		ArchiveName:         o.ArchiveName,
	}
	if l.SharedBlobDirPath == "" {
		l.SharedBlobDirPath = archive.SharedBlobDir
	}
	l.common = newCommon(&o.CommonOpts)

	return l
}

func (l *Loader) copy(ctx context.Context) {
	l.common.initErrorHandler(ctx)
	l.common.initWorker(ctx, l.worker)
}

func (l *Loader) newLoadCacheDir() (string, error) {
	cd, err := os.MkdirTemp(archive.CacheDir(), "*")
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	logrus.Debugf("create save cache dir: %v", cd)
	return cd, nil
}

func (l *Loader) loadIndex() error {
	// TODO:
	return nil
}

// Run loads images from tarball archive to destination image registry
func (s *Loader) Run(ctx context.Context) error {
	ar, err := archive.NewReader(s.ArchiveName)
	if err != nil {
		return fmt.Errorf("failed to create archive %q: %w", s.ArchiveName, err)
	}
	s.ar = ar

	s.copy(ctx)
	if len(s.failedImageList) != 0 {
		return fmt.Errorf("some images failed to load")
	}
	return nil
}

func (l *Loader) worker(ctx context.Context, id int) {
	defer l.common.waitGroup.Done()
	for {
		select {
		case <-ctx.Done():
			logrus.WithFields(logrus.Fields{"w": id}).
				Debugf("worker stopped: %v", ctx.Err())
			return
		case obj, ok := <-l.common.objectCh:
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
				l.common.errorCh <- err
				l.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
				cancel()
				continue
			}
			err = obj.source.Copy(copyContext, obj.destination, l.common.imageSpecSet)
			if err != nil {
				l.common.errorCh <- err
				l.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
				cancel()
				continue
			}

			// images loaded from cache folder
			l.arMutex.Lock()
			l.arMutex.Unlock()

			// Cancel the context after copy image
			cancel()
		}
	}
}
