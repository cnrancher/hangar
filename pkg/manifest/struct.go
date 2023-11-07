package manifest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ManifestImages []*ManifestImage

type ManifestImage struct {
	size      int64
	digest    digest.Digest
	mediaType string
	platform  manifestPlatform
}

func NewManifestImage(
	ctx context.Context, referenceName string, sysContext *types.SystemContext,
) (*ManifestImage, error) {
	inspector, err := NewInspector(ctx, &InspectorOption{
		ReferenceName: referenceName,
		SystemContext: sysContext,
	})
	if err != nil {
		return nil, err
	}
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
		size:      int64(len(b)),
		digest:    digest,
		mediaType: mime,
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

type manifestPlatform struct {
	arch       string
	os         string
	variant    string
	osVersion  string
	osFeatures []string
}

func (p *ManifestImage) Equal(d *ManifestImage) bool {
	if p == nil || d == nil {
		return false
	}
	if p.digest != d.digest {
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
	for _, p := range images {
		if p.Equal(d) {
			return true
		}
	}
	return false
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
