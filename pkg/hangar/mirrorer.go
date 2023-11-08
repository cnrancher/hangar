package hangar

import (
	"context"
	"fmt"
	"time"

	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/manifest"
	"github.com/cnrancher/hangar/pkg/source"
	"github.com/cnrancher/hangar/pkg/types"
	"github.com/cnrancher/hangar/pkg/utils"
	imagetypes "github.com/containers/image/v5/types"
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

func NewMirrorer(o *MirrorerOpts) *Mirrorer {
	m := &Mirrorer{
		SourceRegistry:      o.SourceRegistry,
		DestinationRegistry: o.DestinationRegistry,
		SourceProject:       o.SourceProject,
		DestinationProject:  o.DestinationProject,
	}
	m.common = newCommon(&o.CommonOpts)
	return m
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
		}
		if err != nil {
			m.common.recordFailedImage(line)
			m.handleError(err)
			continue
		}
		object.id = i + 1
		m.handleObject(object)
	}

	close(m.common.objectCh)
	// Waiting for all images copied
	m.common.waitGroup.Wait()
	close(m.common.errorCh)
	// Waiting for all error messages were handled properly
	m.common.errorWaitGroup.Wait()
}

// Run mirror images from source to destination registry.
func (m *Mirrorer) Run(ctx context.Context) error {
	m.copy(ctx)
	if len(m.failedImageSet) != 0 {
		for i := range m.failedImageSet {
			fmt.Printf("%v\n", i)
		}
		return fmt.Errorf("some images failed to mirror")
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
		Type:     types.TypeDocker,
		Registry: sourceRegistry,
		Project:  sourceProject,
		Name:     utils.GetImageName(line),
		Tag:      utils.GetImageTag(line),
		SystemContext: &imagetypes.SystemContext{
			DockerInsecureSkipTLSVerify: imagetypes.NewOptionalBool(m.common.tlsVerify),
			OCIInsecureSkipTLSVerify:    m.common.tlsVerify,
		},
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
		Type:     types.TypeDocker,
		Registry: m.DestinationRegistry,
		Project:  destProject,
		Name:     utils.GetImageName(line),
		Tag:      utils.GetImageTag(line),
		SystemContext: &imagetypes.SystemContext{
			DockerInsecureSkipTLSVerify: imagetypes.NewOptionalBool(m.common.tlsVerify),
			OCIInsecureSkipTLSVerify:    m.common.tlsVerify,
		},
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
		Type:     types.TypeDocker,
		Registry: sourceRegistry,
		Project:  sourceProject,
		Name:     utils.GetImageName(spec[0]),
		Tag:      spec[2],
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
		Type:     types.TypeDocker,
		Registry: m.DestinationRegistry,
		Project:  destProject,
		Name:     utils.GetImageName(spec[1]),
		Tag:      spec[2],
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
			m.handleError(err)
			m.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
		}
	}()

	err = obj.source.Init(copyContext)
	if err != nil {
		return
	}
	err = obj.destination.Init(copyContext)
	if err != nil {
		return
	}
	logrus.WithFields(logrus.Fields{
		"IMG": obj.id,
	}).Infof("Copying  [%v] => [%v]",
		obj.source.ReferenceNameWithoutTransport(),
		obj.destination.ReferenceNameWithoutTransport())
	err = obj.source.Copy(copyContext, obj.destination, m.common.imageSpecSet)
	if err != nil {
		return
	}

	builder, err := manifest.NewBuilder(&manifest.BuilderOpts{
		ReferenceName: obj.destination.ReferenceName(),
		SystemContext: &imagetypes.SystemContext{
			DockerInsecureSkipTLSVerify: imagetypes.NewOptionalBool(m.common.tlsVerify),
			OCIInsecureSkipTLSVerify:    m.common.tlsVerify,
		},
	})
	if err != nil {
		return
	}
	copiedImage := obj.source.GetCopiedImage()
	if len(copiedImage.Images) == 0 {
		return
	}
	for _, image := range copiedImage.Images {
		var mi *manifest.ManifestImage
		mi, err = manifest.NewManifestImage(ctx,
			fmt.Sprintf("docker://%s@%s", copiedImage.Source, image.Digest), nil)
		if err != nil {
			err = fmt.Errorf("failed to create manifest image: %w", err)
		}
		builder.Add(mi)
	}
	if err = builder.Push(ctx); err != nil {
		err = fmt.Errorf("failed to push manifest: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"IMG": obj.id,
	}).Infof("Mirrored [%v] => [%v]",
		obj.source.ReferenceNameWithoutTransport(),
		obj.destination.ReferenceNameWithoutTransport())
}
