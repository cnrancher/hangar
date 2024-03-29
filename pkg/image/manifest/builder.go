package manifest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cnrancher/hangar/pkg/image/internal/private"

	"github.com/containers/common/pkg/retry"
	manifestv5 "github.com/containers/image/v5/manifest"
	alltransportsv5 "github.com/containers/image/v5/transports/alltransports"
	typesv5 "github.com/containers/image/v5/types"
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
	if b.images.Contains(p) {
		return
	}
	if i := b.images.FindPlatformIndex(&p.platform); i >= 0 {
		b.images = append(b.images[:i], b.images[i+1:]...)
	}
	b.images = append(b.images, p)
}

func (b *Builder) Images() int {
	return len(b.images)
}

func (b *Builder) Push(ctx context.Context) error {
	if len(b.images) == 0 {
		return fmt.Errorf("manifest builder: no images added to builder")
	}
	list := manifestv5.Schema2List{
		SchemaVersion: 2,
		MediaType:     manifestv5.DockerV2ListMediaType,
		Manifests:     make([]manifestv5.Schema2ManifestDescriptor, 0),
	}

	for _, img := range b.images {
		s2desc := manifestv5.Schema2ManifestDescriptor{
			Schema2Descriptor: manifestv5.Schema2Descriptor{
				MediaType: img.MediaType,
				Size:      img.Size,
				Digest:    img.Digest,
			},
			Platform: manifestv5.Schema2PlatformSpec{
				Architecture: img.platform.arch,
				OS:           img.platform.os,
				Variant:      img.platform.variant,
				OSVersion:    img.platform.osVersion,
				OSFeatures:   img.platform.osFeatures,
			},
		}
		list.Manifests = append(list.Manifests, s2desc)
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
