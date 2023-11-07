package manifest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

// Builder is the builder to build DockerV2ListMediaType manifest.
type Builder struct {
	// dest image reference name
	name string
	// dest image reference
	reference types.ImageReference
	// images
	images ManifestImages
	// systemContext
	systemContext *types.SystemContext
}

type BuilderOpts struct {
	ReferenceName string
	SystemContext *types.SystemContext
}

func NewBuilder(o *BuilderOpts) (*Builder, error) {
	ref, err := alltransports.ParseImageName(o.ReferenceName)
	if err != nil {
		return nil, err
	}

	b := &Builder{
		name:          o.ReferenceName,
		reference:     ref,
		images:        nil,
		systemContext: o.SystemContext,
	}
	if b.systemContext == nil {
		b.systemContext = &types.SystemContext{}
	}
	return b, nil
}

func (b *Builder) Add(p *ManifestImage) {
	if b.images.Contains(p) {
		return
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
	list := manifest.Schema2List{
		SchemaVersion: 2,
		MediaType:     manifest.DockerV2ListMediaType,
		Manifests:     make([]manifest.Schema2ManifestDescriptor, 0),
	}

	for _, img := range b.images {
		s2desc := manifest.Schema2ManifestDescriptor{
			Schema2Descriptor: manifest.Schema2Descriptor{
				MediaType: img.mediaType,
				Size:      int64(img.size),
				Digest:    img.digest,
			},
			Platform: manifest.Schema2PlatformSpec{
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

	dest, err := b.reference.NewImageDestination(ctx, b.systemContext)
	if err != nil {
		return fmt.Errorf("manifest builder: %w", err)
	}
	err = dest.PutManifest(ctx, d, nil)
	return nil
}
