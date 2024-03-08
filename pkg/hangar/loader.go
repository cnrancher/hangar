package hangar

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/harbor"
	"github.com/cnrancher/hangar/pkg/image/destination"
	"github.com/cnrancher/hangar/pkg/image/manifest"
	"github.com/cnrancher/hangar/pkg/image/source"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"

	"github.com/containers/image/v5/pkg/docker/config"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
)

// loadObject is the object sending to worker pool when loading image
type loadObject struct {
	image   *archive.Image
	timeout time.Duration
	id      int
}

// Loader loads images from hangar archive file to registry server.
type Loader struct {
	*common

	// ar is the archive reader.
	ar *archive.Reader
	// arMutex is the mutex for decompress archive files
	arMutex *sync.Mutex
	// index is the archive index.
	index *archive.Index
	// indexImageSet is map[image name]*archive.Image .
	indexImageSet map[string]*archive.Image
	// layerManager manages the layers
	layerManager *layerManager

	// Specify the source image registry.
	SourceRegistry string
	// Specify the source image project.
	SourceProject string
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

	// Specify the source image registry.
	SourceRegistry string
	// Specify the source image project.
	SourceProject string
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

func NewLoader(o *LoaderOpts) (*Loader, error) {
	l := &Loader{
		ar:            nil,
		arMutex:       &sync.Mutex{},
		index:         archive.NewIndex(),
		indexImageSet: make(map[string]*archive.Image),
		layerManager:  nil,

		SourceRegistry:      o.SourceRegistry,
		SourceProject:       o.SourceProject,
		DestinationRegistry: o.DestinationRegistry,
		DestinationProject:  o.DestinationProject,
		Directory:           o.Directory,
		SharedBlobDirPath:   o.SharedBlobDirPath,
		ArchiveName:         o.ArchiveName,
	}
	if l.SharedBlobDirPath == "" {
		l.SharedBlobDirPath = archive.SharedBlobDir
	}
	var err error
	l.common, err = newCommon(&o.CommonOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create common: %w", err)
	}

	l.ar, err = archive.NewReader(l.ArchiveName)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive reader: %w", err)
	}
	b, err := l.ar.Index()
	if err != nil {
		return nil, fmt.Errorf("ar.Index: %w", err)
	}
	err = l.index.Unmarshal(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal index data: %w", err)
	}
	if len(l.index.List) == 0 {
		logrus.Warnf("No images in %q", o.ArchiveName)
	}
	for i := 0; i < len(l.index.List); i++ {
		source := l.index.List[i].Source
		tag := l.index.List[i].Tag
		l.indexImageSet[source+":"+tag] = l.index.List[i]
	}
	lm, err := newLayerManager(l.index)
	if err != nil {
		return nil, fmt.Errorf("failed to init layer manager: %w", err)
	}
	l.layerManager = lm

	return l, nil
}

func (l *Loader) copy(ctx context.Context) {
	l.common.initErrorHandler(ctx)
	l.common.initWorker(ctx, l.worker)
	if len(l.common.images) > 0 {
		// Load images according to image list specified by user.
		for i, line := range l.common.images {
			switch imagelist.Detect(line) {
			case imagelist.TypeDefault:
			default:
				logrus.Warnf("Ignore image list line %q: invalid format", line)
				continue
			}
			registry := utils.GetRegistryName(line)
			if l.SourceRegistry != "" {
				registry = l.SourceRegistry
			}
			project := utils.GetProjectName(line)
			if l.SourceProject != "" {
				project = l.SourceProject
			}
			name := utils.GetImageName(line)
			tag := utils.GetImageTag(line)
			imageName := fmt.Sprintf("%s/%s/%s:%s",
				registry, project, name, tag)
			image, ok := l.indexImageSet[imageName]
			if !ok {
				l.recordFailedImage(line)
				l.handleError(
					NewError(
						i+1,
						fmt.Errorf("image [%v] not exists in archive", imageName),
						nil,
						nil,
					),
				)
				continue
			}
			object := &loadObject{
				id:    i + 1,
				image: image,
			}
			l.handleObject(object)
		}
	} else {
		// Load all images from archive file.
		for i, image := range l.index.List {
			object := &loadObject{
				id:    i + 1,
				image: image,
			}
			l.handleObject(object)
		}
	}
	l.waitWorkers()
	l.layerManager.cleanAll()
	if err := l.ar.Close(); err != nil {
		logrus.Errorf("failed to close archive reader: %v", err)
	}
}

// Run loads images from hangar archive to destination image registry
func (l *Loader) Run(ctx context.Context) error {
	if err := l.initHarborProject(ctx); err != nil {
		// Harbor Project error should not block the loading process
		// since users can create the Harbor Project manually if failed.
		logrus.Warnf("Failed to init Harbor Project: %v", err)
		logrus.Warnf("Please create the Harbor Project manually.")
	}
	l.copy(ctx)
	if len(l.failedImageSet) != 0 {
		v := make([]string, 0, len(l.failedImageSet))
		for i := range l.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Copy failed image list: \n%v", strings.Join(v, "\n"))
		return ErrCopyFailed
	}
	return nil
}

func (l *Loader) initHarborProject(ctx context.Context) error {
	harborURL, err := harbor.GetURL(ctx, l.DestinationRegistry,
		!l.systemContext.OCIInsecureSkipTLSVerify)
	if err != nil {
		if errors.Is(err, harbor.ErrRegistryIsNotHarbor) {
			return nil
		}
		return err
	}
	credential, err := config.GetCredentials(nil, l.DestinationRegistry)
	if err != nil {
		return fmt.Errorf("failed to get credential of %q: %w",
			l.DestinationRegistry, err)
	}

	projectSet := map[string]bool{}
	if len(l.DestinationProject) > 0 {
		projectSet[l.DestinationProject] = true
	}
	for i := 0; len(l.DestinationProject) == 0 && i < len(l.index.List); i++ {
		project := utils.GetProjectName(l.index.List[i].Source)
		projectSet[project] = true
	}
	for project := range projectSet {
		exists, err := harbor.ProjectExists(
			ctx, project, harborURL, &credential,
			!l.systemContext.OCIInsecureSkipTLSVerify)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		err = harbor.CreateProject(
			ctx, project, harborURL, &credential,
			!l.systemContext.OCIInsecureSkipTLSVerify)
		if err != nil {
			return err
		}
		logrus.Infof("Created Harbor V2 project %q for registry %q",
			project, l.DestinationRegistry)
	}
	return nil
}

func (l *Loader) worker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*loadObject)
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
	imageName := obj.image.Source + ":" + obj.image.Tag
	// Use defer to handle error message.
	defer func() {
		if err != nil {
			l.handleError(NewError(obj.id, err, nil, nil))
			l.recordFailedImage(imageName)
		}
		cancel()
	}()

	// Init destination image spec.
	destinationRegistry := utils.GetRegistryName(imageName)
	if l.DestinationRegistry != "" {
		destinationRegistry = l.DestinationRegistry
	}
	destinationProject := utils.GetProjectName(imageName)
	if l.DestinationProject != "" {
		destinationProject = l.DestinationProject
	}
	dest, err := destination.NewDestination(&destination.Option{
		Type:          types.TypeDocker,
		Registry:      destinationRegistry,
		Project:       destinationProject,
		Name:          utils.GetImageName(imageName),
		Tag:           obj.image.Tag,
		SystemContext: l.systemContext,
	})
	if err != nil {
		err = fmt.Errorf("failed to create destination image: %w", err)
		return
	}
	if err = dest.Init(copyContext); err != nil {
		err = fmt.Errorf("failed to init destination image: %w", err)
		return
	}

	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("Loading [%v] => [%v]",
			imageName, dest.ReferenceNameWithoutTransport())
	if obj.image.IsSigstoreSignature() {
		logrus.WithFields(logrus.Fields{"IMG": obj.id}).
			Warnf("Failed to load image [%v]: %v, "+
				"use 'hangar sign' to sign the image instead of copy the signature directly",
				imageName, utils.ErrIsSigstoreSignature)
		err = utils.ErrIsSigstoreSignature
		return
	}
	var manifestImages = make(manifest.Images, 0)
	for _, img := range obj.image.Images {
		if img.Digest == "" {
			logrus.WithFields(logrus.Fields{"IMG": obj.id}).
				Warnf("Skip invalid image [%v] [%v] [%v]",
					imageName, img.Arch, img.OS)
			continue
		}

		var (
			tmpDir string
			imgRef string
		)
		imgRef = dest.ReferenceNameDigest(img.Digest)
		l.arMutex.Lock()
		tmpDir, err = l.ar.DecompressImageTmp(&img, l.common.imageSpecSet)
		l.arMutex.Unlock()
		// Register defer function to clean-up cache.
		defer func(d string, img archive.ImageSpec) {
			if d != "" {
				if err := os.RemoveAll(d); err != nil {
					logrus.Warnf("Failed to cleanup [%v]: %v", d, err)
				}
			}
			l.layerManager.clean(&img, &l.imageSpecSet)
		}(tmpDir, img)

		if err != nil {
			if !errors.Is(err, utils.ErrNoAvailableImage) {
				err = fmt.Errorf("failed to decompress image [%v]: %w", imgRef, err)
				return
			}
			refName := fmt.Sprintf("%s@%s", obj.image.Source, img.Digest)
			if img.OSVersion != "" {
				logrus.WithFields(logrus.Fields{"IMG": obj.id}).
					Infof("Skip [%s] [%s%s] [%s] [%s]",
						refName, img.Arch, img.Variant, img.OS, img.OSVersion)
			} else {
				logrus.WithFields(logrus.Fields{"IMG": obj.id}).
					Infof("Skip [%s] [%s%s] [%s]",
						refName, img.Arch, img.Variant, img.OS)
			}
			err = nil
			continue
		}

		l.arMutex.Lock()
		err = l.layerManager.decompressLayer(&img, l.ar)
		l.arMutex.Unlock()
		if err != nil {
			err = fmt.Errorf("arch [%v] os [%v]: %w", img.Arch, img.OS, err)
			return
		}

		var src *source.Source
		src, err = source.NewSource(&source.Option{
			Type:      types.TypeOci,
			Directory: tmpDir,
			SystemContext: utils.SystemContextWithSharedBlobDir(
				l.systemContext, l.layerManager.sharedBlobDir()),
		})
		if err != nil {
			err = fmt.Errorf("failed to create source image: %w", err)
			return
		}
		if err = src.Init(copyContext); err != nil {
			err = fmt.Errorf("failed to init [%v]: %w",
				src.ReferenceName(), err)
			return
		}
		err = src.Copy(copyContext, &source.CopyOptions{
			RemoveSignatures:   false,
			SigstorePrivateKey: l.common.sigstorePrivateKey,
			SigstorePassphrase: l.common.sigstorePassphrase,
			Destination:        dest,
			Set:                l.common.imageSpecSet,
			Policy:             l.common.policy,
		})
		if err != nil {
			if errors.Is(err, utils.ErrNoAvailableImage) {
				logrus.WithFields(logrus.Fields{"IMG": obj.id}).
					Warnf("Skip loading image [%v]: %v", imageName, err)
				err = nil
				return
			}
			err = fmt.Errorf("failed to copy [%v] to [%v]: %w",
				src.ReferenceName(), dest.ReferenceName(), err)
			return
		}

		var mi *manifest.Image
		mi, err = manifest.NewImageByInspect(
			copyContext, dest.ReferenceNameDigest(img.Digest), dest.SystemContext(),
		)
		if err != nil {
			err = fmt.Errorf("failed to create manifest image: %w", err)
			return
		}
		mi.UpdatePlatform(
			img.Arch, img.Variant, img.OS, img.OSVersion, img.OSFeatures)
		manifestImages = append(manifestImages, mi)
	}

	destManifestImages := dest.ManifestImages()
	if len(destManifestImages) > 0 {
		// If no new image copied to destination registry, skip re-create
		// manifest index for destination image.
		var skipBuildManifest = true
		for _, img := range manifestImages {
			if !destManifestImages.ContainDigest(img.Digest) {
				skipBuildManifest = false
				break
			}
		}
		if skipBuildManifest {
			logrus.Debugf("skip build manifest for image [%v]: already exists",
				dest.ReferenceName())
			return
		}
	}

	// Init manifest Builder.
	builder, err := manifest.NewBuilder(&manifest.BuilderOpts{
		ReferenceName: dest.ReferenceName(),
		SystemContext: dest.SystemContext(),
	})
	if err != nil {
		err = fmt.Errorf("failed to create manifest builder: %w", err)
		return
	}
	// Add images already exists on destination registry into builder firstly.
	for _, img := range destManifestImages {
		builder.Add(img)
	}
	// Then add new copied images to builder, update existing images.
	for _, img := range manifestImages {
		builder.Add(img)
	}
	if builder.Images() == 0 {
		err = fmt.Errorf("failed to load [%v]: some images failed to load", imageName)
		return
	}
	if err = builder.Push(ctx); err != nil {
		err = fmt.Errorf("failed to push manifest: %w", err)
		return
	}
}

func (l *Loader) Validate(ctx context.Context) error {
	l.validate(ctx)
	if len(l.failedImageSet) != 0 {
		v := make([]string, 0, len(l.failedImageSet))
		for i := range l.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Validate failed image list: \n%v", strings.Join(v, "\n"))
		return ErrValidateFailed
	}
	return nil
}

func (l *Loader) validate(ctx context.Context) {
	l.common.initErrorHandler(ctx)
	l.common.initWorker(ctx, l.validateWorker)
	if len(l.common.images) > 0 {
		// Validate images according to image list specified by user.
		for i, line := range l.common.images {
			registry := utils.GetRegistryName(line)
			if l.SourceRegistry != "" {
				registry = l.SourceRegistry
			}
			project := utils.GetProjectName(line)
			if l.SourceProject != "" {
				project = l.SourceProject
			}
			name := utils.GetImageName(line)
			tag := utils.GetImageTag(line)
			imageName := fmt.Sprintf("%s/%s/%s:%s",
				registry, project, name, tag)
			image, ok := l.indexImageSet[imageName]
			if !ok {
				l.recordFailedImage(line)
				l.handleError(
					NewError(
						i+1,
						fmt.Errorf("image [%v] not exists in archive", imageName),
						nil,
						nil,
					),
				)
				continue
			}
			object := &loadObject{
				id:    i + 1,
				image: image,
			}
			l.handleObject(object)
		}
	} else {
		// Validate all images from archive file.
		for i, image := range l.index.List {
			object := &loadObject{
				id:    i + 1,
				image: image,
			}
			l.handleObject(object)
		}
	}
	l.waitWorkers()
	l.layerManager.cleanAll()
	if err := l.ar.Close(); err != nil {
		logrus.Errorf("failed to close archive reader: %v", err)
	}
}

func (l *Loader) validateWorker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*loadObject)
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
	imageName := obj.image.Source + ":" + obj.image.Tag
	// Use defer to handle error message.
	defer func() {
		cancel()
		if err != nil {
			l.handleError(NewError(obj.id, err, nil, nil))
			l.recordFailedImage(imageName)
		}
	}()
	logrus.Debugf("Validating [%v]", imageName)

	// Init source image.
	if len(obj.image.Images) == 0 {
		return
	}
	sourceDigestSet := map[digest.Digest]bool{}
	for _, img := range obj.image.Images {
		if len(l.imageSpecSet["arch"]) > 0 && !l.imageSpecSet["arch"][img.Arch] {
			continue
		}
		if len(l.imageSpecSet["os"]) > 0 && !l.imageSpecSet["os"][img.OS] {
			continue
		}
		sourceDigestSet[img.Digest] = true
	}
	if len(sourceDigestSet) == 0 {
		return
	}

	// Init destination image.
	destinationRegistry := utils.GetRegistryName(imageName)
	if l.DestinationRegistry != "" {
		destinationRegistry = l.DestinationRegistry
	}
	destinationProject := utils.GetProjectName(imageName)
	if l.DestinationProject != "" {
		destinationProject = l.DestinationProject
	}
	dest, err := destination.NewDestination(&destination.Option{
		Type:          types.TypeDocker,
		Registry:      destinationRegistry,
		Project:       destinationProject,
		Name:          utils.GetImageName(imageName),
		Tag:           obj.image.Tag,
		SystemContext: l.systemContext,
	})
	if err != nil {
		err = fmt.Errorf("failed to create destination image: %w", err)
		return
	}
	if err = dest.Init(validateContext); err != nil {
		err = fmt.Errorf("failed to init destination image: %w", err)
		return
	}
	if !dest.Exists() {
		logrus.WithFields(logrus.Fields{"IMG": obj.id}).
			Errorf("Image [%v] does not exists in destination registry server",
				dest.ReferenceNameWithoutTransport())
		err = fmt.Errorf("FAILED: [%v]", imageName)
		return
	}
	destImage := dest.ImageBySet(l.imageSpecSet)
	destDigestSet := map[digest.Digest]bool{}
	for _, img := range destImage.Images {
		destDigestSet[img.Digest] = true
	}
	for d := range sourceDigestSet {
		if !destDigestSet[d] {
			logrus.WithFields(logrus.Fields{"IMG": obj.id}).
				Errorf("Image [%v] digest [%v] does not exists in destination registry",
					dest.ReferenceNameWithoutTransport(), d)
			err = fmt.Errorf("FAILED: [%v]", imageName)
			return
		}
	}

	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("PASS: [%v]", imageName)
}
