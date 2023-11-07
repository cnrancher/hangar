package manifest

import (
	"context"
	"fmt"

	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Inspector provides similar functions of 'skopeo inspect' command.
type Inspector struct {
	// reference name
	name          string
	systemContext *types.SystemContext
	source        types.ImageSource
	mime          string
}

type InspectorOption struct {
	// Reference of the image to be inspected (Optional)
	Reference types.ImageReference
	// ReferenceName of the image (Optional)
	ReferenceName string
	// SystemContext pointer, can be nil.
	SystemContext *types.SystemContext
}

func NewInspector(ctx context.Context, o *InspectorOption) (*Inspector, error) {
	var (
		ref           types.ImageReference = o.Reference
		systemContext *types.SystemContext = o.SystemContext
		err           error
	)
	if ref == nil {
		ref, err = alltransports.ParseImageName(o.ReferenceName)
		if err != nil {
			return nil, err
		}
	}
	if systemContext == nil {
		systemContext = &types.SystemContext{}
	}
	source, err := ref.NewImageSource(ctx, systemContext)
	if err != nil {
		return nil, err
	}
	_, mime, err := source.GetManifest(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &Inspector{
		name:          o.ReferenceName,
		systemContext: systemContext,
		source:        source,
		mime:          mime,
	}, nil
}

func (ins *Inspector) Close() error {
	return ins.source.Close()
}

func (ins *Inspector) Raw(ctx context.Context) ([]byte, string, error) {
	return ins.source.GetManifest(ctx, nil)
}

func (ins *Inspector) Config(ctx context.Context) ([]byte, error) {
	if ins.mime == manifest.DockerV2ListMediaType ||
		ins.mime == imgspecv1.MediaTypeImageIndex {
		return nil, fmt.Errorf("unable to get config from mime %q", ins.mime)
	}
	image, err := image.FromUnparsedImage(
		ctx, ins.systemContext, image.UnparsedInstance(ins.source, nil))
	if err != nil {
		return nil, err
	}
	return image.ConfigBlob(ctx)
}

func (ins *Inspector) ConfigInfo(ctx context.Context) (*types.BlobInfo, error) {
	if ins.mime == manifest.DockerV2ListMediaType ||
		ins.mime == imgspecv1.MediaTypeImageIndex {
		return nil, fmt.Errorf(
			"unable to get ConfigInfo from mime %q", ins.mime)
	}
	img, err := image.FromUnparsedImage(
		ctx, ins.systemContext, image.UnparsedInstance(ins.source, nil))
	if err != nil {
		return nil, err
	}
	blobInfo := img.ConfigInfo()
	return &blobInfo, nil
}

func (ins *Inspector) Inspect(ctx context.Context) (*types.ImageInspectInfo, error) {
	if ins.mime == manifest.DockerV2ListMediaType ||
		ins.mime == imgspecv1.MediaTypeImageIndex {
		return nil, fmt.Errorf("unable to inspect mime %q", ins.mime)
	}
	image, err := image.FromUnparsedImage(
		ctx, ins.systemContext, image.UnparsedInstance(ins.source, nil))
	if err != nil {
		return nil, err
	}
	return image.Inspect(ctx)
}
