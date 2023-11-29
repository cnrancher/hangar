package hangar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/manifest"
	"github.com/cnrancher/hangar/pkg/source"
	"github.com/cnrancher/hangar/pkg/types"
	"github.com/cnrancher/hangar/pkg/utils"
	imagemanifest "github.com/containers/image/v5/manifest"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
)

// mirrorObject is the object sending to worker pool when copying image
type mirrorObject struct {
	image       string
	source      *source.Source
	destination *destination.Destination
	timeout     time.Duration
	id          int
}

// Mirrorer mirrors multipule images between image registries.
type Mirrorer struct {
	*common

	// Override the registry of source image to be copied
	SourceRegistry string
	// Override the registry of the copied destination image
	DestinationRegistry string
	// Override the project of source image to be copied
	SourceProject string
	// Override the project of the copied destination image
	DestinationProject string
}

type MirrorerOpts struct {
	CommonOpts

	SourceRegistry      string
	DestinationRegistry string
	SourceProject       string
	DestinationProject  string
}

func NewMirrorer(o *MirrorerOpts) (*Mirrorer, error) {
	m := &Mirrorer{
		SourceRegistry:      o.SourceRegistry,
		DestinationRegistry: o.DestinationRegistry,
		SourceProject:       o.SourceProject,
		DestinationProject:  o.DestinationProject,
	}
	var err error
	m.common, err = newCommon(&o.CommonOpts)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Mirrorer) copy(ctx context.Context) {
	m.common.initErrorHandler(ctx)
	m.common.initWorker(ctx, m.worker)
	for i, line := range m.common.images {
		var (
			object *mirrorObject
			err    error
		)
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
			object, err = m.mirrorObjectImageListTypeDefault(line)
		case imagelist.TypeMirror:
			object, err = m.mirrorObjectImageListTypeMirror(line)
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", line)
			continue
		}
		if err != nil {
			m.common.recordFailedImage(line)
			m.handleError(err)
			continue
		}
		object.id = i + 1
		m.handleObject(object)
	}
	m.waitWorkers()
}

// Run mirror images from source to destination registry.
func (m *Mirrorer) Run(ctx context.Context) error {
	m.copy(ctx)
	if len(m.failedImageSet) != 0 {
		v := make([]string, 0, len(m.failedImageSet))
		for i := range m.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Copy failed image list: \n%v", strings.Join(v, "\n"))
		return ErrCopyFailed
	}
	return nil
}

func (m *Mirrorer) mirrorObjectImageListTypeDefault(line string) (*mirrorObject, error) {
	object := &mirrorObject{
		image: line,
	}
	sourceRegistry := utils.GetRegistryName(line)
	if m.SourceRegistry != "" {
		sourceRegistry = m.SourceRegistry
	}
	sourceProject := utils.GetProjectName(line)
	if m.SourceProject != "" {
		sourceProject = m.SourceProject
	}
	src, err := source.NewSource(&source.Option{
		Type:          types.TypeDocker,
		Registry:      sourceRegistry,
		Project:       sourceProject,
		Name:          utils.GetImageName(line),
		Tag:           utils.GetImageTag(line),
		SystemContext: m.systemContext,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init source image: %v", err)
	}
	object.source = src
	destProject := utils.GetProjectName(line)
	if m.DestinationProject != "" {
		destProject = m.DestinationProject
	}
	dest, err := destination.NewDestination(&destination.Option{
		Type:          types.TypeDocker,
		Registry:      m.DestinationRegistry,
		Project:       destProject,
		Name:          utils.GetImageName(line),
		Tag:           utils.GetImageTag(line),
		SystemContext: m.systemContext,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init dest image: %v", err)
	}
	object.destination = dest
	return object, nil
}

func (m *Mirrorer) mirrorObjectImageListTypeMirror(line string) (*mirrorObject, error) {
	object := &mirrorObject{
		image: line,
	}

	spec, _ := imagelist.GetMirrorSpec(line)
	if len(spec) != 3 {
		return nil, fmt.Errorf("ignore line %q in image list: invalid format", line)
	}
	sourceRegistry := utils.GetRegistryName(spec[0])
	if m.SourceRegistry != "" {
		sourceRegistry = m.SourceRegistry
	}
	sourceProject := utils.GetProjectName(spec[0])
	if m.SourceProject != "" {
		sourceProject = m.SourceProject
	}
	src, err := source.NewSource(&source.Option{
		Type:          types.TypeDocker,
		Registry:      sourceRegistry,
		Project:       sourceProject,
		Name:          utils.GetImageName(spec[0]),
		Tag:           spec[2],
		SystemContext: m.systemContext,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init source image: %v", err)
	}
	object.source = src
	destProject := utils.GetProjectName(spec[1])
	if m.DestinationProject != "" {
		destProject = m.DestinationProject
	}
	dest, err := destination.NewDestination(&destination.Option{
		Type:          types.TypeDocker,
		Registry:      m.DestinationRegistry,
		Project:       destProject,
		Name:          utils.GetImageName(spec[1]),
		Tag:           spec[2],
		SystemContext: m.systemContext,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init dest image: %v", err)
	}
	object.destination = dest
	return object, nil
}

func (m *Mirrorer) worker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*mirrorObject)
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
			m.handleError(fmt.Errorf("error occurred when copy [%v] to [%v]: %w",
				obj.source.ReferenceNameWithoutTransport(),
				obj.destination.ReferenceNameWithoutTransport(), err))
			m.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
		}
	}()

	err = obj.source.Init(copyContext)
	if err != nil {
		err = fmt.Errorf("failed to init [%v]: %w",
			obj.source.ReferenceName(), err)
		return
	}
	err = obj.destination.Init(copyContext)
	if err != nil {
		err = fmt.Errorf("failed to init [%v]: %w",
			obj.destination.ReferenceName(), err)
		return
	}
	logrus.WithFields(logrus.Fields{
		"IMG": obj.id,
	}).Infof("Copying [%v] => [%v]",
		obj.source.ReferenceNameWithoutTransport(),
		obj.destination.ReferenceNameWithoutTransport())
	err = obj.source.Copy(copyContext, obj.destination, m.imageSpecSet, m.policy)
	if err != nil {
		if errors.Is(err, utils.ErrNoAvailableImage) {
			logrus.WithFields(logrus.Fields{"IMG": obj.id}).
				Warnf("Skip copy image [%v]: %v",
					obj.source.ReferenceNameWithoutTransport(), err)
			err = nil
		} else {
			return
		}
	}

	copiedImage := obj.source.GetCopiedImage()
	if len(copiedImage.Images) == 0 {
		return
	}
	var manifestImages = make(manifest.ManifestImages, 0)
	for _, image := range copiedImage.Images {
		var mi *manifest.ManifestImage
		mi, err = manifest.NewManifestImageByInspect(
			copyContext,
			obj.destination.ReferenceNameDigest(image.Digest),
			obj.destination.SystemContext(),
		)
		if err != nil {
			err = fmt.Errorf("failed to create manifest image: %w", err)
			return
		}
		mi.UpdatePlatform(
			image.Arch, image.Variant, image.OS, image.OSVersion, image.OSFeatures)
		manifestImages = append(manifestImages, mi)
	}
	destManifestImages := obj.destination.ManifestImages()
	if len(destManifestImages) > 0 {
		// If no new image copied to the destination registry, skip re-create
		// manifest index for destination image.
		var skipBuildManifest bool = true
		for _, img := range destManifestImages {
			if !manifestImages.ContainDigest(img.Digest) {
				skipBuildManifest = false
				break
			}
		}
		if skipBuildManifest {
			logrus.Debugf("skip build manifest for image [%v]: already exists",
				obj.destination.ReferenceName())
			return
		}
	}

	builder, err := manifest.NewBuilder(&manifest.BuilderOpts{
		ReferenceName: obj.destination.ReferenceName(),
		SystemContext: obj.destination.SystemContext(),
	})
	if err != nil {
		err = fmt.Errorf("failed to create mafiest builder: %w", err)
		return
	}
	// Merge new added images with destination manifest index.
	for _, img := range manifestImages {
		builder.Add(img)
	}
	for _, img := range destManifestImages {
		builder.Add(img)
	}
	if builder.Images() == 0 {
		return
	}
	if err = builder.Push(ctx); err != nil {
		err = fmt.Errorf("failed to push manifest: %w", err)
		return
	}
}

func (m *Mirrorer) Validate(ctx context.Context) error {
	m.validate(ctx)
	if len(m.failedImageSet) != 0 {
		v := make([]string, 0, len(m.failedImageSet))
		for i := range m.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Copy failed image list: \n%v", strings.Join(v, "\n"))
		return ErrCopyFailed
	}
	return nil
}

func (m *Mirrorer) validate(ctx context.Context) {
	m.common.initErrorHandler(ctx)
	m.initWorker(ctx, m.validateWorker)
	for i, line := range m.common.images {
		var (
			object *mirrorObject
			err    error
		)
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
			object, err = m.mirrorObjectImageListTypeDefault(line)
		case imagelist.TypeMirror:
			object, err = m.mirrorObjectImageListTypeMirror(line)
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", line)
			continue
		}
		if err != nil {
			m.common.recordFailedImage(line)
			m.handleError(err)
			continue
		}
		object.id = i + 1
		m.handleObject(object)
	}
	m.waitWorkers()
}

func (m *Mirrorer) validateWorker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*mirrorObject)
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
			m.handleError(NewError(obj.id, err, obj.source, obj.destination))
			m.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
		}
	}()
	err = obj.source.Init(validateContext)
	if err != nil {
		return
	}
	err = obj.destination.Init(validateContext)
	if err != nil {
		return
	}
	if !obj.destination.Exists() {
		logrus.WithFields(logrus.Fields{"IMG": obj.id}).
			Errorf("[%v] does not exists",
				obj.destination.ReferenceNameWithoutTransport())
		err = fmt.Errorf("FAILED: [%v] != [%v]",
			obj.source.ReferenceNameWithoutTransport(),
			obj.destination.ReferenceNameWithoutTransport())
		return
	}

	switch obj.source.MIME() {
	case imagemanifest.DockerV2Schema1MediaType,
		imagemanifest.DockerV2Schema1SignedMediaType:
		// Could not compare image digest since the destination mediaType
		// was changed during copy.
	default:
		destImages := obj.destination.ImageBySet(m.imageSpecSet)
		destDigestSet := map[digest.Digest]bool{}
		for _, img := range destImages.Images {
			destDigestSet[img.Digest] = true
		}
		sourceImages := obj.source.ImageBySet(m.imageSpecSet)
		for _, img := range sourceImages.Images {
			if !destDigestSet[img.Digest] {
				logrus.WithFields(logrus.Fields{"IMG": obj.id}).
					Errorf("Image [%v] does not exists in destination registry",
						obj.destination.ReferenceNameDigest(img.Digest))
				err = fmt.Errorf("FAILED: [%v] != [%v]",
					obj.source.ReferenceNameWithoutTransport(),
					obj.destination.ReferenceNameWithoutTransport())
				return
			}
		}
	}

	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("PASS: [%v] == [%v]",
			obj.source.ReferenceNameWithoutTransport(),
			obj.destination.ReferenceNameWithoutTransport())
}
