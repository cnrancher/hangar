package manifest

import (
	"context"

	"github.com/cnrancher/hangar/pkg/image/internal/private"
	"github.com/containers/common/pkg/retry"
	imagev5 "github.com/containers/image/v5/image"
	alltransportsv5 "github.com/containers/image/v5/transports/alltransports"
	typesv5 "github.com/containers/image/v5/types"
)

// Inspector provides similar functions of 'skopeo inspect' command.
type Inspector struct {
	// reference name
	name          string
	systemContext *typesv5.SystemContext
	source        typesv5.ImageSource
	retryOpts     *retry.Options
}

type InspectorOption struct {
	// Reference of the image to be inspected (Optional)
	Reference typesv5.ImageReference
	// ReferenceName of the image (Optional)
	ReferenceName string
	// SystemContext pointer, can be nil.
	SystemContext *typesv5.SystemContext
	// The number of times to possibly retry.
	RetryOpts *retry.Options
}

func NewInspector(ctx context.Context, o *InspectorOption) (*Inspector, error) {
	var (
		ref           typesv5.ImageReference = o.Reference
		systemContext *typesv5.SystemContext = o.SystemContext
		err           error
	)
	if ref == nil {
		ref, err = alltransportsv5.ParseImageName(o.ReferenceName)
		if err != nil {
			return nil, err
		}
	}
	if systemContext == nil {
		systemContext = &typesv5.SystemContext{}
	}
	p := &Inspector{
		name:          o.ReferenceName,
		systemContext: systemContext,
		source:        nil,
		retryOpts:     o.RetryOpts,
	}
	if p.retryOpts == nil {
		p.retryOpts = private.RetryOptions()
	}

	var source typesv5.ImageSource
	retry.IfNecessary(ctx, func() error {
		// NewImageSource requires network connection
		source, err = ref.NewImageSource(ctx, systemContext)
		return err
	}, p.retryOpts)
	if err != nil {
		return nil, err
	}
	p.source = source

	return p, nil
}

func (p *Inspector) Close() error {
	return p.source.Close()
}

func (p *Inspector) Raw(ctx context.Context) ([]byte, string, error) {
	var (
		b    []byte
		mime string
		err  error
	)
	if err = retry.IfNecessary(ctx, func() error {
		b, mime, err = p.source.GetManifest(ctx, nil)
		return err
	}, p.retryOpts); err != nil {
		return nil, "", err
	}
	return b, mime, nil
}

func (p *Inspector) Config(ctx context.Context) ([]byte, error) {
	var (
		img typesv5.Image
		err error
	)
	if err = retry.IfNecessary(ctx, func() error {
		img, err = imagev5.FromUnparsedImage(
			ctx, p.systemContext, imagev5.UnparsedInstance(p.source, nil))
		return err
	}, p.retryOpts); err != nil {
		return nil, err
	}
	return img.ConfigBlob(ctx)
}

func (p *Inspector) ConfigInfo(ctx context.Context) (*typesv5.BlobInfo, error) {
	var (
		img typesv5.Image
		err error
	)
	if err = retry.IfNecessary(ctx, func() error {
		img, err = imagev5.FromUnparsedImage(
			ctx, p.systemContext, imagev5.UnparsedInstance(p.source, nil))
		return err
	}, p.retryOpts); err != nil {
		return nil, err
	}
	blobInfo := img.ConfigInfo()
	return &blobInfo, nil
}

func (p *Inspector) Inspect(ctx context.Context) (*typesv5.ImageInspectInfo, error) {
	image, err := imagev5.FromUnparsedImage(
		ctx, p.systemContext, imagev5.UnparsedInstance(p.source, nil))
	if err != nil {
		return nil, err
	}
	var (
		info *typesv5.ImageInspectInfo
	)
	if err = retry.IfNecessary(ctx, func() error {
		var err error
		info, err = image.Inspect(ctx)
		return err
	}, p.retryOpts); err != nil {
		return nil, err
	}
	return info, nil
}
