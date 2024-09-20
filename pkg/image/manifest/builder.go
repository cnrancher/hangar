package manifest

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/cnrancher/hangar/pkg/image/internal/private"

	"github.com/containers/common/pkg/retry"
	alltransportsv5 "github.com/containers/image/v5/transports/alltransports"
	typesv5 "github.com/containers/image/v5/types"
	imgspec "github.com/opencontainers/image-spec/specs-go"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Builder is the builder to build DockerV2ListMediaType manifest.
type Builder struct {
	// dest image reference name
	name string
	// dest image reference
	reference typesv5.ImageReference
	// images
	images Images
	// systemContext
	systemContext *typesv5.SystemContext

	retryOpts *retry.Options
}

type BuilderOpts struct {
	ReferenceName string
	SystemContext *typesv5.SystemContext
	// RetryOpts is the options to retry on error, can be nil
	RetryOpts *retry.Options
}

func NewBuilder(o *BuilderOpts) (*Builder, error) {
	ref, err := alltransportsv5.ParseImageName(o.ReferenceName)
	if err != nil {
		return nil, err
	}

	b := &Builder{
		name:          o.ReferenceName,
		reference:     ref,
		images:        nil,
		systemContext: o.SystemContext,
		retryOpts:     o.RetryOpts,
	}
	if b.systemContext == nil {
		b.systemContext = &typesv5.SystemContext{}
	}
	if o.RetryOpts == nil {
		b.retryOpts = private.RetryOptions()
	}
	return b, nil
}

func (b *Builder) Add(p *Image) {
	// Replace if digest already exists
	for i, img := range b.images {
		if img.Digest != p.Digest {
			continue
		}

		// The image digest maybe same on SLSA manifest attestation.
		if len(img.Annotations) != 0 || len(p.Annotations) != 0 {
			if !reflect.DeepEqual(img.Annotations, p.Annotations) {
				continue
			}
		}
		b.images = append(b.images[:i], b.images[i+1:]...)
	}
	b.images = append(b.images, p.DeepCopy())
}

func (b *Builder) Images() int {
	return len(b.images)
}

func (b *Builder) Push(ctx context.Context) error {
	if len(b.images) == 0 {
		return fmt.Errorf("manifest builder: no images added to builder")
	}
	// list := manifestv5.Schema2List{
	// 	SchemaVersion: 2,
	// 	MediaType:     manifestv5.DockerV2ListMediaType,
	// 	Manifests:     make([]manifestv5.Schema2ManifestDescriptor, 0),
	// }
	list := imgspecv1.Index{
		Versioned: imgspec.Versioned{
			SchemaVersion: 2,
		},
		MediaType: imgspecv1.MediaTypeImageIndex,
		Manifests: make([]imgspecv1.Descriptor, 0),
	}

	for _, img := range b.images {
		m := imgspecv1.Descriptor{
			MediaType:   img.MediaType,
			Size:        img.Size,
			Digest:      img.Digest,
			Annotations: img.Annotations,
			Platform: &imgspecv1.Platform{
				Architecture: img.platform.arch,
				OS:           img.platform.os,
				OSVersion:    img.platform.osVersion,
				OSFeatures:   img.platform.osFeatures,
				Variant:      img.platform.variant,
			},
		}
		list.Manifests = append(list.Manifests, m)
	}
	d, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("manifest builder: %w", err)
	}

	var (
		dest typesv5.ImageDestination
	)
	if err = retry.IfNecessary(ctx, func() error {
		dest, err = b.reference.NewImageDestination(ctx, b.systemContext)
		if err != nil {
			if dest != nil {
				dest.Close()
			}
			return err
		}
		return nil
	}, b.retryOpts); err != nil {
		return fmt.Errorf("manifest builder: %w", err)
	}
	defer dest.Close()
	if err = retry.IfNecessary(ctx, func() error {
		return dest.PutManifest(ctx, d, nil)
	}, b.retryOpts); err != nil {
		return fmt.Errorf("manifest builder: %w", err)
	}
	return nil
}
