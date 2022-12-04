package image

import (
	"fmt"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

// Imagerer interface is the specific image
type Imagerer interface {
	// Source gets the source image
	Source() string

	// Destination gets the destination image
	Destination() string

	// Arch gets the architecture of the image
	Arch() string

	// OS gets the OS of the image
	OS() string

	// Digest gets the digest of the image,
	// return empty string if not set
	Digest() string

	// SetDigest sets the digest of the image
	SetDigest(string)

	// Copy executes the copy operation of the image
	Copy() error

	// Copied checks the image is copied or not
	Copied() bool
}

type Image struct {
	source      string
	destination string
	tag         string
	arch        string
	variant     string
	os          string
	digest      string
	copied      bool

	sourceSchemaVersion int
	sourceMediaType     string
}

type ImageOptions struct {
	Source      string
	Destination string
	Tag         string
	Arch        string
	Variant     string
	OS          string
	Digest      string

	SourceSchemaVersion int
	SourceMediaType     string
}

func NewImage(opts *ImageOptions) *Image {
	return &Image{
		source:              opts.Source,
		destination:         opts.Destination,
		tag:                 opts.Tag,
		arch:                opts.Arch,
		variant:             opts.Variant,
		os:                  opts.OS,
		digest:              opts.Digest,
		sourceSchemaVersion: opts.SourceSchemaVersion,
		sourceMediaType:     opts.SourceMediaType,
	}
}

func (img *Image) Source() string {
	return img.source
}

func (img *Image) Destination() string {
	return img.destination
}

func (img *Image) Arch() string {
	return img.arch
}

func (img *Image) OS() string {
	return img.os
}

func (img *Image) Digest() string {
	return img.digest
}

func (img *Image) SetDigest(d string) {
	img.digest = d
}

func (img *Image) Copy() error {
	if img == nil {
		return u.ErrNilPointer
	}

	if img.source == "" || img.destination == "" || img.arch == "" {
		return u.ErrInvalidParameter
	}

	if err := img.copyIfChanged(); err != nil {
		return fmt.Errorf("Copy: %w", err)
	}
	img.copied = true
	return nil
}

func (img *Image) Copied() bool {
	return img.copied
}

func (img *Image) copyIfChanged() error {
	var (
		srcDockerImage string
		dstDockerImage string
	)

	switch img.sourceSchemaVersion {
	case 1:
		// docker://registry/repository:va.b.c
		srcDockerImage = fmt.Sprintf("docker://%s:%s", img.source, img.tag)
	case 2:
		switch img.sourceMediaType {
		case u.MediaTypeManifestListV2:
			// docker://registry/repository@sha256:abcdef...
			srcDockerImage = fmt.Sprintf("docker://%s@%s",
				img.source, img.digest)
		case u.MediaTypeManifestV2:
			// docker://registry/repository:va.b.c
			srcDockerImage = fmt.Sprintf("docker://%s:%s", img.source, img.tag)
		}
	}

	// TODO: handle the image variant
	// docker://registry/repository:va.b.c-ARCH
	dstDockerImage = fmt.Sprintf("docker://%s:%s-%s",
		img.destination, img.tag, img.arch)

	// Inspect the source image info
	sourceManifestBuff, err := registry.SkopeoInspect(srcDockerImage, "--raw")
	if err != nil {
		// if source image not found, return error.
		return fmt.Errorf("copyIfChanged failed inspect source image: %w", err)
	}
	// logrus.Debug("sourceManifest: ", sourceManifestBuff.String())

	destManifestBuff, err := registry.SkopeoInspect(dstDockerImage, "--raw")
	if err != nil {
		// if destination image not found, set destManifestBuff to nil
		destManifestBuff = nil
	}
	// logrus.Debug("destManifest: ", destManifestBuff.String())

	srcManifestSum := "sha256:" + u.Sha256Sum(sourceManifestBuff.String())
	dstManifestSum := "sha256:" + u.Sha256Sum(destManifestBuff.String())
	// compare the source manifest with the dest manifest
	if srcManifestSum == dstManifestSum {
		logrus.Infof("Unchanged: %s == %s", srcDockerImage, dstDockerImage)
		logrus.Debugf("source digest: %s", srcManifestSum)
		logrus.Debugf("destin digest: %s", dstManifestSum)
		return nil
	}
	logrus.Infof("Copying: %s => %s", srcDockerImage, dstDockerImage)
	return registry.SkopeoCopyArchOS(
		img.arch, img.os, srcDockerImage, dstDockerImage)
}
