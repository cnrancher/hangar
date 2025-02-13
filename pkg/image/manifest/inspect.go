package manifest

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cnrancher/hangar/pkg/image/internal/private"
	"github.com/containers/common/pkg/retry"
	imagev5 "github.com/containers/image/v5/image"
	"github.com/containers/image/v5/pkg/blobinfocache/none"
	alltransportsv5 "github.com/containers/image/v5/transports/alltransports"
	typesv5 "github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
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

func (p *Inspector) Provenance(ctx context.Context) ([]byte, error) {
	var (
		b   []byte
		img typesv5.Image
		err error
	)
	if err = retry.IfNecessary(ctx, func() error {
		img, err = imagev5.FromUnparsedImage(
			ctx, p.systemContext, imagev5.UnparsedInstance(p.source, nil))
		layers := img.LayerInfos()
		for _, l := range layers {
			if len(l.Annotations) == 0 {
				logrus.Debugf("skip non-provenance layer %v", l.Digest)
				continue
			}
			var t string
			for k, v := range l.Annotations {
				if !strings.Contains(k, "intoto") {
					continue
				}
				if !strings.Contains(k, "predicate") {
					continue
				}
				t = v
			}
			if strings.Contains(t, "slsa") {
				rc, _, err := p.source.GetBlob(ctx, l, none.NoCache)
				if err != nil {
					return fmt.Errorf("failed to get blob: %w", err)
				}
				b, err = io.ReadAll(rc)
				rc.Close()
				if err != nil {
					return fmt.Errorf("failed to read blob: %w", err)
				}
				return nil
			}
		}
		return fmt.Errorf("SLSA provenance data not found in image")
	}, p.retryOpts); err != nil {
		return nil, fmt.Errorf("inspector.Provenance: %w", err)
	}
	return b, nil
}

func (p *Inspector) SBOM(ctx context.Context) ([]byte, error) {
	var (
		b   []byte
		img typesv5.Image
		err error
	)
	if err = retry.IfNecessary(ctx, func() error {
		img, err = imagev5.FromUnparsedImage(
			ctx, p.systemContext, imagev5.UnparsedInstance(p.source, nil))
		layers := img.LayerInfos()
		for _, l := range layers {
			if len(l.Annotations) == 0 {
				logrus.Debugf("skip non-provenance layer %v", l.Digest)
				continue
			}
			var t string
			for k, v := range l.Annotations {
				if !strings.Contains(k, "intoto") {
					continue
				}
				if !strings.Contains(k, "predicate") {
					continue
				}
				t = v
			}
			if strings.Contains(t, "bom") {
				rc, _, err := p.source.GetBlob(ctx, l, none.NoCache)
				if err != nil {
					return fmt.Errorf("failed to get blob: %w", err)
				}
				b, err = io.ReadAll(rc)
				rc.Close()
				if err != nil {
					return fmt.Errorf("failed to read blob: %w", err)
				}
				return nil
			}
		}
		return fmt.Errorf("SBOM data not found in image")
	}, p.retryOpts); err != nil {
		return nil, fmt.Errorf("inspector.SBOM: %w", err)
	}
	return b, nil
}
