package manifest

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"

	manifestv5 "github.com/containers/image/v5/manifest"
	typesv5 "github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Images []*Image

type Image struct {
	Size        int64
	Digest      digest.Digest
	MediaType   string
	Annotations map[string]string // OCI image index v1 supports annotations
	platform    manifestPlatform
}

func NewImageByInspect(
	ctx context.Context, referenceName string, sysContext *typesv5.SystemContext,
) (*Image, error) {
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
	case manifestv5.DockerV2ListMediaType, imgspecv1.MediaTypeImageIndex:
		return nil, fmt.Errorf("unsupoorted to add %q to manifest builder", mime)
	}
	digest, err := manifestv5.Digest(b)
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
	mi := &Image{
		Size:        int64(len(b)),
		Digest:      digest,
		MediaType:   mime,
		Annotations: map[string]string{},
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

func NewImage(
	digest digest.Digest, mime string, size int64, annotations map[string]string,
) *Image {
	mi := &Image{
		Digest:      digest,
		MediaType:   mime,
		Size:        size,
		Annotations: map[string]string{},
	}
	if len(annotations) != 0 {
		mi.Annotations = annotations
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

func (p *Image) SetArch(arch string) {
	p.platform.arch = arch
}

func (p *Image) SetOS(os string) {
	p.platform.os = os
}

func (p *Image) SetVariant(variant string) {
	p.platform.variant = variant
}

func (p *Image) SetOsVersion(v string) {
	p.platform.osVersion = v
}

func (p *Image) SetOsFeature(v []string) {
	p.platform.osFeatures = slices.Clone(v)
}

func (p *Image) UpdatePlatform(
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

func (p *Image) Equal(d *Image) bool {
	if p == nil || d == nil {
		return false
	}
	if p.Digest != d.Digest {
		return false
	}
	if len(p.Annotations) != 0 || len(d.Annotations) != 0 {
		if !reflect.DeepEqual(p.Annotations, d.Annotations) {
			return false
		}
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

func (p *Image) DeepCopy() *Image {
	if p == nil {
		return nil
	}
	return &Image{
		Size:        p.Size,
		Digest:      p.Digest,
		MediaType:   p.MediaType,
		Annotations: maps.Clone(p.Annotations),
		platform: manifestPlatform{
			arch:       p.platform.arch,
			os:         p.platform.os,
			variant:    p.platform.variant,
			osVersion:  p.platform.osVersion,
			osFeatures: slices.Clone(p.platform.osFeatures),
		},
	}
}

func (images Images) Contains(d *Image) bool {
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

func (images Images) ContainDigest(d digest.Digest) bool {
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

func (images Images) FindDigestIndex(p *Image) int {
	if len(images) == 0 || p == nil {
		return -1
	}
	for i, img := range images {
		if img.Digest == p.Digest {
			return i
		}
	}
	return -1
}

func (images Images) FindPlatformIndex(p *Image) int {
	if len(images) == 0 || p == nil {
		return -1
	}
	for i, img := range images {
		if img.platform.equal(&p.platform) {
			return i
		}
	}
	return -1
}

func (images Images) Equal(d Images) bool {
	if len(images) != len(d) {
		return false
	}
	for i := 0; i < len(images); i++ {
		if !images[i].Equal(d[i]) {
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
