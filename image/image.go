package image

import "fmt"

// Imager interface is the specific image
type Imager interface {
	// Source gets the source image
	Source() string

	// Destination gets the destination image
	Destination() string

	// Arch gets the architecture of the image
	Arch() string

	// Variant gets the variant of the image
	Variant() string

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

	// Save saves the image into local directory
	Save() error

	// Saved checks the image is saved or not
	Saved() bool

	Load() error

	Loaded() bool

	// SetID sets the ID of the Imager
	SetID(string)

	// ID gets the ID of the Imager
	ID() string
}

type Image struct {
	source      string
	destination string
	tag         string
	arch        string
	variant     string
	os          string
	digest      string // digest is the source image manifest sha256sum
	directory   string

	copied bool
	saved  bool
	loaded bool

	sourceSchemaVersion int
	sourceMediaType     string

	// ID of the Imager
	iID string
	// ID of the Mirrorer
	mID string
}

type ImageOptions struct {
	Source      string
	Destination string
	Tag         string
	Arch        string
	Variant     string
	OS          string
	Digest      string
	Directory   string

	SourceSchemaVersion int
	SourceMediaType     string

	MID string
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
		directory:           opts.Directory,
		sourceSchemaVersion: opts.SourceSchemaVersion,
		sourceMediaType:     opts.SourceMediaType,
		mID:                 opts.MID,
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

func (img *Image) Variant() string {
	return img.variant
}

func (img *Image) Digest() string {
	return img.digest
}

func (img *Image) SetDigest(d string) {
	img.digest = d
}

func (img *Image) SetID(id string) {
	// format: 01, 02, 03...
	img.iID = id
}

func (img *Image) ID() string {
	return img.iID
}

// CopiedTag gets the tag of the copied image,
// the format should be: ${VERSION}-${ARCH}${VARIANT}
func CopiedTag(tag, arch, variant string) string {
	switch arch {
	case "amd64":
		return fmt.Sprintf("%s-%s", tag, arch)
	case "arm64":
		// there is only one variant of arm64 is v8, so discard it
		return fmt.Sprintf("%s-%s", tag, arch)
	case "arm":
		// arm has variant v5, v7, etc...
		return fmt.Sprintf("%s-%s%s", tag, arch, variant)
	default:
		// other arch: s390x, ppc64...
		return fmt.Sprintf("%s-%s%s", tag, arch, variant)
	}
}
