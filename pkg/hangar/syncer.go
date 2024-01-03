package hangar

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/source"
	"github.com/cnrancher/hangar/pkg/types"
	"github.com/cnrancher/hangar/pkg/utils"
	imagemanifest "github.com/containers/image/v5/manifest"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
)

// syncObject is the object for sending to worker pool when syncing image
type syncObject struct {
	image       string
	source      *source.Source
	destination *destination.Destination
	timeout     time.Duration
	id          int
}

type Syncer struct {
	*common

	au        *archive.Updater
	auMutex   *sync.RWMutex
	index     *archive.Index
	layersSet map[digest.Digest]bool

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

func NewSyncer(o *SyncerOpts) (*Syncer, error) {
	s := &Syncer{
		auMutex:   &sync.RWMutex{},
		index:     archive.NewIndex(),
		layersSet: make(map[digest.Digest]bool),

		SourceRegistry:    o.SourceRegistry,
		SourceProject:     o.SourceProject,
		SharedBlobDirPath: o.SharedBlobDirPath,
		ArchiveName:       o.ArchiveName,
	}
	if s.SharedBlobDirPath == "" {
		s.SharedBlobDirPath = archive.SharedBlobDir
	}
	var err error
	s.common, err = newCommon(&o.CommonOpts)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Syncer) copy(ctx context.Context) {
	s.common.initErrorHandler(ctx)
	s.common.initWorker(ctx, s.worker)
	for i, img := range s.common.images {
		switch imagelist.Detect(img) {
		case imagelist.TypeDefault:
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", img)
			continue
		}
		object := &syncObject{
			id:    i + 1,
			image: img,
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
			SystemContext: s.systemContext,
		})
		if err != nil {
			s.handleError(fmt.Errorf("failed to init source image: %w", err))
			s.recordFailedImage(img)
			continue
		}
		object.source = src

		cd, err := s.newSaveCacheDir()
		if err != nil {
			s.handleError(fmt.Errorf("failed to create cache dir: %w", err))
			s.recordFailedImage(img)
			continue
		}
		sd := path.Join(cd, s.SharedBlobDirPath)
		dest, err := destination.NewDestination(&destination.Option{
			Type:      types.TypeOci,
			Directory: cd,
			Name:      utils.GetImageName(img),
			Tag:       utils.GetImageTag(img),
			SystemContext: utils.SystemContextWithSharedBlobDir(
				s.systemContext, sd),
		})
		if err != nil {
			s.handleError(fmt.Errorf("failed to init dest image: %w", err))
			os.RemoveAll(cd)
			s.recordFailedImage(img)
			continue
		}
		object.destination = dest
		if err = s.handleObject(object); err != nil {
			os.RemoveAll(cd)
		}
	}
	s.waitWorkers()
	if err := s.updateIndex(); err != nil {
		logrus.Errorf("failed to write index file: %v", err)
	}
	if err := s.au.Close(); err != nil {
		logrus.Errorf("failed to close archive updater: %v", err)
	}
}

func (s *Syncer) newSaveCacheDir() (string, error) {
	cd, err := os.MkdirTemp(utils.CacheDir(), "*")
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	logrus.Debugf("create save cache dir: %v", cd)
	return cd, nil
}

func (s *Syncer) updateIndex() error {
	s.au.SetIndex(s.index)
	return s.au.UpdateIndex()
}

// Run append images from registry server into local directory / hangar archive.
func (s *Syncer) Run(ctx context.Context) error {
	// Init Archive Updater.
	au, err := archive.NewUpdater(s.ArchiveName)
	if err != nil {
		return fmt.Errorf("failed to open archive %q: %w", s.ArchiveName, err)
	}
	s.au = au
	s.index = au.Index()
	// Init layerSet.
	for _, images := range s.index.List {
		for _, spec := range images.Images {
			for _, layer := range spec.Layers {
				s.layersSet[layer] = true
			}
			s.layersSet[spec.Digest] = true
			if spec.Config != "" {
				s.layersSet[spec.Config] = true
			}
		}
	}

	s.copy(ctx)
	if len(s.failedImageSet) != 0 {
		v := make([]string, 0, len(s.failedImageSet))
		for i := range s.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Sync failed image list: \n%v", strings.Join(v, "\n"))
		return ErrCopyFailed
	}
	return nil
}

func (s *Syncer) worker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*syncObject)
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
		if err != nil {
			s.handleError(NewError(obj.id, err, obj.source, obj.destination))
			s.recordFailedImage(obj.image)
		}
		cancel()
		// Delete cache dir.
		if err = os.RemoveAll(obj.destination.Directory()); err != nil {
			err = fmt.Errorf("failed to delete cache dir %q: %w",
				obj.destination.Directory(), err)
		}
	}()

	err = obj.source.Init(copyContext)
	if err != nil {
		err = fmt.Errorf("failed to init source: %w", err)
		return
	}
	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("Syncing [%v]", obj.source.ReferenceNameWithoutTransport())
	err = obj.destination.Init(copyContext)
	if err != nil {
		err = fmt.Errorf("failed to init destination: %w", err)
		return
	}
	err = obj.source.Copy(copyContext, obj.destination, s.imageSpecSet, s.policy)
	if err != nil {
		if errors.Is(err, utils.ErrNoAvailableImage) {
			logrus.WithFields(logrus.Fields{"IMG": obj.id}).
				Warnf("Skip copy image [%v]: %v",
					obj.source.ReferenceNameWithoutTransport(), err)
			err = nil
		} else {
			err = fmt.Errorf("failed to copy [%v] to [%v]: %w",
				obj.source.ReferenceName(), obj.destination.ReferenceName(), err)
			return
		}
	}

	// Images copied to cache folder, write to archive file.
	s.auMutex.Lock()
	defer s.auMutex.Unlock()

	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Debugf("Compressing [%v]", obj.destination.ReferenceNameWithoutTransport())

	destDir := obj.destination.ReferenceNameWithoutTransport()
	copiedImage := obj.source.GetCopiedImage()
	imageBlobs := map[digest.Digest]bool{}
	filesToDelete := map[string]bool{}
	// Record image layers and remove duplicated layers from shared blob dir.
	for _, image := range copiedImage.Images {
		for _, layer := range image.Layers {
			imageBlobs[layer] = true
		}
		if image.Config != "" {
			imageBlobs[image.Config] = true
		}
		imageBlobs[image.Digest] = true
		if s.layersSet[image.Digest] {
			// The image already exists in archive, delete OCI image directory.
			d := path.Join(destDir, image.Digest.Encoded())
			filesToDelete[d] = true
		}
	}
	for blob := range imageBlobs {
		if s.layersSet[blob] {
			d := path.Join(destDir, archive.SharedBlobDir,
				string(blob.Algorithm()), blob.Encoded())
			filesToDelete[d] = true
		} else {
			s.layersSet[blob] = true
		}
	}

	for f := range filesToDelete {
		if _, err := os.Stat(f); err != nil {
			logrus.Warnf("failed to clean duplicated file %q: stat: %v",
				f, err)
		}
		if err := os.RemoveAll(f); err != nil {
			logrus.Warnf("failed to clean duplicated file %q: remove all: %v",
				f, err)
		}
	}

	err = s.au.Append(obj.destination.ReferenceNameWithoutTransport())
	if err != nil {
		err = fmt.Errorf("failed to append files into zip archive: %w", err)
		return
	}
	s.index.Append(copiedImage)
}

func (s *Syncer) Validate(ctx context.Context) error {
	ar, err := archive.NewReader(s.ArchiveName)
	if err != nil {
		return fmt.Errorf("failed to create archive reader: %w", err)
	}
	b, err := ar.Index()
	if err != nil {
		return fmt.Errorf("failed to read archive index: %w", err)
	}
	if err := ar.Close(); err != nil {
		logrus.Errorf("failed to close archive reader: %v", err)
	}
	if err := s.index.Unmarshal(b); err != nil {
		return fmt.Errorf("failed to read archive index: %w", err)
	}

	s.validate(ctx)
	if len(s.failedImageSet) != 0 {
		v := make([]string, 0, len(s.failedImageSet))
		for i := range s.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Validate failed image list: \n%v", strings.Join(v, "\n"))
		return ErrValidateFailed
	}
	return nil
}

func (s *Syncer) validate(ctx context.Context) {
	s.common.initErrorHandler(ctx)
	s.common.initWorker(ctx, s.validateWorker)
	for i, img := range s.common.images {
		switch imagelist.Detect(img) {
		case imagelist.TypeDefault:
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", img)
			continue
		}
		object := &syncObject{
			id:    i + 1,
			image: img,
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
			SystemContext: s.systemContext,
		})
		if err != nil {
			s.handleError(fmt.Errorf("failed to init source image: %w", err))
			s.recordFailedImage(img)
			continue
		}
		object.source = src
		s.handleObject(object)
	}
	s.waitWorkers()
}

func (s *Syncer) validateWorker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*syncObject)
	if !ok {
		logrus.Errorf("skip object type(%T), data %v", o, o)
		return
	}

	var (
		validateContext context.Context
		cancel          context.CancelFunc
		err             error
	)
	if obj.timeout > 0 {
		validateContext, cancel = context.WithTimeout(ctx, obj.timeout)
	} else {
		validateContext, cancel = context.WithCancel(ctx)
	}

	defer func() {
		cancel()
		if err != nil {
			s.handleError(NewError(obj.id, err, nil, nil))
			s.recordFailedImage(obj.image)
		}
	}()

	err = obj.source.Init(validateContext)
	if err != nil {
		return
	}
	var fail bool
	switch obj.source.MIME() {
	case imagemanifest.DockerV2Schema1MediaType,
		imagemanifest.DockerV2Schema1SignedMediaType:
		// Could not compare image digest since the destination mediaType
		// was changed during copy.
		if !s.index.HasReference(
			obj.source.Project(), obj.source.Name(), obj.source.Tag()) {
			fail = true
		}
	default:
		image := obj.source.ImageBySet(s.imageSpecSet)
		if !s.index.Has(image) {
			fail = true
		}
	}

	if fail {
		logrus.WithFields(logrus.Fields{"IMG": obj.id}).
			Errorf("Image [%v] does not exists in archive index",
				obj.source.ReferenceNameWithoutTransport())
		err = fmt.Errorf("FAILED: [%v]",
			obj.source.ReferenceNameWithoutTransport())
		return
	}

	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("PASS: [%v]", obj.source.ReferenceNameWithoutTransport())
}
