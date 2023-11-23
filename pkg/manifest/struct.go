package manifest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/utils/strings/slices"
)

type ManifestImages []*ManifestImage

type ManifestImage struct {
	Size      int64
	Digest    digest.Digest
	MediaType string
	platform  manifestPlatform
}

func NewManifestImageByInspect(
	ctx context.Context, referenceName string, sysContext *types.SystemContext,
) (*ManifestImage, error) {
	inspector, err := NewInspector(ctx, &InspectorOption{
		ReferenceName: referenceName,
		SystemContext: sysContext,
	})
	if err != nil {
		return nil, err
	}
	defer inspector.Close()

	b, mime, err := inspector.Raw(ctx)
	if err != nil {
		return nil, err
	}
	switch mime {
	case manifest.DockerV2ListMediaType, imgspecv1.MediaTypeImageIndex:
		return nil, fmt.Errorf("unsupoorted to add %q to manifest builder", mime)
	}
	digest, err := manifest.Digest(b)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate image digest: %w", err)
	}
	cb, err := inspector.Config(ctx)
	if err != nil {
		return nil, err
	}
	config := &imgspecv1.Image{}
	err = json.Unmarshal(cb, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get image config: %w", err)
	}
	mi := &ManifestImage{
		Size:      int64(len(b)),
		Digest:    digest,
		MediaType: mime,
		platform: manifestPlatform{
			arch:       config.Architecture,
			os:         config.OS,
			variant:    config.Variant,
			osVersion:  config.OSVersion,
			osFeatures: config.OSFeatures,
		},
	}

	return mi, nil
}

func NewManifestImage(
	digest digest.Digest, mime string, size int64,
) *ManifestImage {
	mi := &ManifestImage{
		Digest:    digest,
		MediaType: mime,
		Size:      size,
	}

	return mi
}

type manifestPlatform struct {
	arch       string
	os         string
	variant    string
	osVersion  string
	osFeatures []string
}

func (p *ManifestImage) SetArch(arch string) {
	p.platform.arch = arch
}

func (p *ManifestImage) SetOS(os string) {
	p.platform.os = os
}

func (p *ManifestImage) SetVariant(variant string) {
	p.platform.variant = variant
}

func (p *ManifestImage) SetOsVersion(v string) {
	p.platform.osVersion = v
}

func (p *ManifestImage) SetOsFeature(v []string) {
	p.platform.osFeatures = slices.Clone(v)
}

func (p *ManifestImage) UpdatePlatform(
	arch, variant, os, osVersion string, osFeatures []string,
) {
	p.platform = manifestPlatform{
		arch:       arch,
		variant:    variant,
		os:         os,
		osVersion:  osVersion,
		osFeatures: slices.Clone(osFeatures),
	}
}

func (p *ManifestImage) Equal(d *ManifestImage) bool {
	if p == nil || d == nil {
		return false
	}
	if p.Digest != d.Digest {
		return false
	}
	if p.platform.arch != d.platform.arch {
		return false
	}
	if p.platform.os != d.platform.os {
		return false
	}
	if p.platform.variant != d.platform.variant {
		return false
	}
	if p.platform.osVersion != d.platform.osVersion {
		return false
	}
	if len(p.platform.osFeatures) != len(d.platform.osFeatures) {
		return false
	}
	for i := 0; i < len(p.platform.osFeatures); i++ {
		if p.platform.osFeatures[i] != d.platform.osFeatures[i] {
			return false
		}
	}
	return true
}

func (images ManifestImages) Contains(d *ManifestImage) bool {
	if len(images) == 0 {
		return false
	}
	for _, p := range images {
		if p.Equal(d) {
			return true
		}
	}
	return false
}

func (images ManifestImages) ContainDigest(d digest.Digest) bool {
	if len(images) == 0 {
		return false
	}
	for _, p := range images {
		if p.Digest == d {
			return true
		}
	}
	return false
}

func (images ManifestImages) FindPlatformIndex(p *manifestPlatform) int {
	if len(images) == 0 {
		return -1
	}
	for i, img := range images {
		if img.platform.equal(p) {
			return i
		}
	}
	return -1
}

func (p ManifestImages) Equal(d ManifestImages) bool {
	if len(p) != len(d) {
		return false
	}
	for i := 0; i < len(p); i++ {
		if !p[i].Equal(d[i]) {
			return false
		}
	}
	return true
}

func (p *manifestPlatform) equal(d *manifestPlatform) bool {
	if p.arch != d.arch {
		return false
	}
	if p.os != d.os {
		return false
	}
	if p.variant != d.variant {
		return false
	}
	if p.osVersion != d.osVersion {
		return false
	}
	if len(p.osFeatures) != len(d.osFeatures) {
		return false
	}
	for i := 0; i < len(p.osFeatures); i++ {
		if p.osFeatures[i] != d.osFeatures[i] {
			return false
		}
	}
	return true
}
