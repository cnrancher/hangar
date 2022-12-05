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

	// CopiedTag gets the tag of the copied image:
	// the format should be: ${VERSION}-${ARCH}${VARIANT}
	CopiedTag() string
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

	if img.sourceSchemaVersion == 1 {
		// get digests from copied dest image
		destImage := fmt.Sprintf("docker://%s:%s",
			img.destination, img.CopiedTag())
		// `skopeo inspect docker://docker.io/${repository}:${version}-${arch}`
		out, err := registry.SkopeoInspect(destImage, "--raw")
		if err != nil {
			return fmt.Errorf("Copy: %w", err)
		}
		digest := "sha256:" + u.Sha256Sum(out)
		img.digest = digest
	}

	img.copied = true
	return nil
}

func (img *Image) Copied() bool {
	return img.copied
}

func (img *Image) CopiedTag() string {
	switch img.arch {
	case "amd64":
		return fmt.Sprintf("%s-%s", img.tag, img.arch)
	case "arm64":
		// there is only one variant of arm64 is v8, so discard it
		return fmt.Sprintf("%s-%s", img.tag, img.arch)
	case "arm":
		// arm has variant v5, v7, etc...
		return fmt.Sprintf("%s-%s%s", img.tag, img.arch, img.variant)
	default:
		// other arch: s390x, ppc64...
		return fmt.Sprintf("%s-%s%s", img.tag, img.arch, img.variant)
	}
}

func (img *Image) copyIfChanged() error {
	var (
		srcDockerImage string
		dstDockerImage string
	)

	if img.sourceSchemaVersion == 1 {
		srcDockerImage = fmt.Sprintf("docker://%s:%s", img.source, img.tag)
		dstDockerImage = fmt.Sprintf("docker://%s:%s",
			img.destination, img.CopiedTag())
		logrus.Infof("[%s] is schema v1, no need to compare", srcDockerImage)
		logrus.Infof("Copying: %s => %s", srcDockerImage, dstDockerImage)
		args := []string{"--format=v2s2", "--override-arch=" + img.arch}
		if img.os != "" {
			args = append(args, "--override-os="+img.os)
		}
		return registry.SkopeoCopy(srcDockerImage, dstDockerImage, args...)
	}

	switch img.sourceMediaType {
	case u.MediaTypeManifestListV2:
		// docker://registry/repository@sha256:abcdef...
		srcDockerImage = fmt.Sprintf("docker://%s@%s", img.source, img.digest)
	case u.MediaTypeManifestV2:
		// docker://registry/repository:va.b.c
		srcDockerImage = fmt.Sprintf("docker://%s:%s", img.source, img.tag)
	}

	// docker://registry/repository:${ORIGINAL_TAG}-${ARCH}${VARIANT}
	dstDockerImage = fmt.Sprintf("docker://%s:%s",
		img.destination, img.CopiedTag())

	// Inspect the source image info
	sourceManifest, err := registry.SkopeoInspect(srcDockerImage, "--raw")
	if err != nil {
		// if source image not found, return error.
		return fmt.Errorf("copyIfChanged failed inspect source image: %w", err)
	}
	// logrus.Debug("sourceManifest: ", sourceManifestBuff.String())

	destManifest, err := registry.SkopeoInspect(dstDockerImage, "--raw")
	if err != nil {
		// if destination image not found, set destManifestBuff to nil
		destManifest = ""
	}
	// logrus.Debug("destManifest: ", destManifestBuff.String())

	var srcManifestSum string
	var dstManifestSum string = "<nil>"
	srcManifestSum = "sha256:" + u.Sha256Sum(sourceManifest)
	if destManifest != "" {
		dstManifestSum = "sha256:" + u.Sha256Sum(destManifest)
	}
	// compare the source manifest with the dest manifest
	if srcManifestSum == dstManifestSum {
		logrus.Infof("Unchanged: %s == %s", srcDockerImage, dstDockerImage)
		logrus.Infof("  source digest: %s", srcManifestSum)
		logrus.Infof("  destin digest: %s", dstManifestSum)
		return nil
	} else {
		logrus.Infof("Digest: %s => %s", srcManifestSum, dstManifestSum)
	}
	logrus.Infof("Copying: %s => %s", srcDockerImage, dstDockerImage)
	args := []string{"--format=v2s2", "--override-arch=" + img.arch}
	if img.os != "" {
		args = append(args, "--override-os="+img.os)
	}
	return registry.SkopeoCopy(srcDockerImage, dstDockerImage, args...)
}
