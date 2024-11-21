package manifest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cnrancher/hangar/pkg/image/internal/private"

	"github.com/containers/common/pkg/retry"
	alltransportsv5 "github.com/containers/image/v5/transports/alltransports"
	typesv5 "github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	imgspec "github.com/opencontainers/image-spec/specs-go"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	annotationKeyReferenceDigest    = "vnd.docker.reference.digest"
	annotationKeyReferenceType      = "vnd.docker.reference.type"
	annotationKeyReferenceTypeValue = "attestation-manifest"
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
	if i := b.images.FindPlatformIndex(p); i >= 0 {
		b.images = append(b.images[:i], b.images[i+1:]...)
	}

	if i := b.images.FindSLSAIndex(p); i >= 0 {
		b.images = append(b.images[:i], b.images[i+1:]...)
	}

	if i := b.images.FindDigestIndex(p); i >= 0 {
		if b.images[i].platform.equal(&p.platform) {
			b.images = append(b.images[:i], b.images[i+1:]...)
		}
	}

	b.images = append(b.images, p.DeepCopy())
}

func (b *Builder) Images() int {
	return len(b.images)
}

func (b *Builder) index() *imgspecv1.Index {
	index := &imgspecv1.Index{
		Versioned: imgspec.Versioned{
			SchemaVersion: 2,
		},
		MediaType: imgspecv1.MediaTypeImageIndex,
		Manifests: make([]imgspecv1.Descriptor, 0),
	}
	if len(b.images) == 0 {
		return index
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
		index.Manifests = append(index.Manifests, m)
	}
	return index
}

func (b *Builder) String() (string, error) {
	index := b.index()
	d, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return "", fmt.Errorf("manifest builder: %w", err)
	}
	return string(d), nil
}

func (b *Builder) RemoveUnExistSLSAProvenance() {
	imageSet := map[digest.Digest]bool{}
	provenanceSet := map[digest.Digest]bool{}

	for _, img := range b.images {
		if img.platform.arch == platformUnknown || img.platform.os == platformUnknown {
			provenanceSet[img.Digest] = true
		} else {
			imageSet[img.Digest] = true
		}
	}

	for dig := range provenanceSet {
		index := -1
		for i := range b.images {
			if b.images[i].Digest != dig {
				continue
			}
			index = i
		}
		if index < 0 {
			continue
		}

		a := b.images[index].Annotations
		if len(a) == 0 {
			continue
		}
		d := a[annotationKeyReferenceDigest]
		if d == "" {
			continue
		}
		if imageSet[digest.Digest(d)] {
			continue
		}
		b.images = append(b.images[:index], b.images[index+1:]...)
	}
}

func (b *Builder) Push(ctx context.Context) error {
	// Remove unexist SLSA Provenances.
	b.RemoveUnExistSLSAProvenance()

	s, err := b.String()
	if err != nil {
		return err
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
		return dest.PutManifest(ctx, []byte(s), nil)
	}, b.retryOpts); err != nil {
		return fmt.Errorf("manifest builder: %w", err)
	}
	return nil
}
