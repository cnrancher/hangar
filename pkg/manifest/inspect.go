package manifest

import (
	"context"
	"time"

	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

const (
	defaultRetryTimes = 2
	defaultRetryDelay = time.Millisecond * 100
)

// Inspector provides similar functions of 'skopeo inspect' command.
type Inspector struct {
	// reference name
	name          string
	systemContext *types.SystemContext
	source        types.ImageSource
	mime          string
	maxRetry      int
	delay         time.Duration
}

type InspectorOption struct {
	// Reference of the image to be inspected (Optional)
	Reference types.ImageReference
	// ReferenceName of the image (Optional)
	ReferenceName string
	// SystemContext pointer, can be nil.
	SystemContext *types.SystemContext
	// The number of times to possibly retry.
	MaxRetry int
	// The delay to use between retries, if set.
	Delay time.Duration
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

	ins := &Inspector{
		name:          o.ReferenceName,
		systemContext: systemContext,
		source:        source,
		mime:          mime,
		maxRetry:      o.MaxRetry,
		delay:         o.Delay,
	}
	if o.MaxRetry == 0 {
		ins.maxRetry = defaultRetryTimes
	}
	if o.Delay == 0 {
		ins.delay = defaultRetryDelay
	}
	return ins, nil
}

func (ins *Inspector) Close() error {
	return ins.source.Close()
}

func (ins *Inspector) Raw(ctx context.Context) ([]byte, string, error) {
	var (
		b    []byte
		mime string
		err  error
	)
	if err = retry.IfNecessary(ctx, func() error {
		b, mime, err = ins.source.GetManifest(ctx, nil)
		return err
	}, &retry.Options{
		MaxRetry: ins.maxRetry,
		Delay:    ins.delay,
	}); err != nil {
		return nil, "", err
	}
	return b, mime, nil
}

func (ins *Inspector) Config(ctx context.Context) ([]byte, error) {
	var (
		img types.Image
		err error
	)
	if err = retry.IfNecessary(ctx, func() error {
		img, err = image.FromUnparsedImage(
			ctx, ins.systemContext, image.UnparsedInstance(ins.source, nil))
		return err
	}, &retry.Options{
		MaxRetry: ins.maxRetry,
		Delay:    ins.delay,
	}); err != nil {
		return nil, err
	}
	return img.ConfigBlob(ctx)
}

func (ins *Inspector) ConfigInfo(ctx context.Context) (*types.BlobInfo, error) {
	var (
		img types.Image
		err error
	)
	if err = retry.IfNecessary(ctx, func() error {
		img, err = image.FromUnparsedImage(
			ctx, ins.systemContext, image.UnparsedInstance(ins.source, nil))
		return err
	}, &retry.Options{
		MaxRetry: ins.maxRetry,
		Delay:    ins.delay,
	}); err != nil {
		return nil, err
	}
	blobInfo := img.ConfigInfo()
	return &blobInfo, nil
}

func (ins *Inspector) Inspect(ctx context.Context) (*types.ImageInspectInfo, error) {
	image, err := image.FromUnparsedImage(
		ctx, ins.systemContext, image.UnparsedInstance(ins.source, nil))
	if err != nil {
		return nil, err
	}
	var (
		info *types.ImageInspectInfo
	)
	if err = retry.IfNecessary(ctx, func() error {
		var err error
		info, err = image.Inspect(ctx)
		return err
	}, &retry.Options{
		MaxRetry: ins.maxRetry,
		Delay:    ins.delay,
	}); err != nil {
		return nil, err
	}
	return info, nil
}
