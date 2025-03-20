package oci

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"helm.sh/helm/v3/pkg/registry"

	signaturev5 "github.com/containers/image/v5/signature"
	typesv5 "github.com/containers/image/v5/types"
)

type common struct {
	insecureSkipVerify bool
	systemContext      *typesv5.SystemContext
	policy             *signaturev5.Policy

	cacheDir    string
	imageDir    string
	imageSource string

	manifestDigest digest.Digest
	config         any
	configDigest   digest.Digest
	layerMediaType string
	annotations    map[string]string
	layers         []digest.Digest
	imageTag       string
}

type CommonOpts struct {
	InsecureSkipVerify bool
	SystemContext      *typesv5.SystemContext
	Policy             *signaturev5.Policy
}

func (c *common) CacheDir() string {
	return c.cacheDir
}

func (c *common) ImageDir() string {
	return c.imageDir
}

func (c *common) Source() string {
	return c.imageSource
}

func (c *common) Cleanup() error {
	if c.cacheDir == "" {
		return nil
	}
	return os.RemoveAll(c.cacheDir)
}

func (c *common) constructOCIImage(r io.Reader) error {
	if len(c.annotations) == 0 {
		return fmt.Errorf("annotations not set")
	}
	if c.config == nil {
		return fmt.Errorf("config not initialized")
	}
	if c.layerMediaType == "" {
		return fmt.Errorf("layer mediaType not set")
	}

	cd, err := newFileCacheDir()
	c.cacheDir = cd
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(cd, "tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	defer tmpFile.Close()
	_, err = io.Copy(tmpFile, r)
	if err != nil {
		return fmt.Errorf("failed to write layer %q: %w", tmpFilePath, err)
	}

	tmpFile.Sync()
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file %q to zero: %w", tmpFilePath, err)
	}
	layerDigest, err := digest.SHA256.FromReader(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to calc %q sha256sum: %w", tmpFilePath, err)
	}

	configData, err := json.MarshalIndent(c.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal file config: %w", err)
	}
	c.configDigest = digest.SHA256.FromBytes(configData)
	layerSize, err := tmpFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("failed to seek file %q current size: %w", tmpFilePath, err)
	}

	m := &imgspecv1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: imgspecv1.MediaTypeImageManifest,
		Config: imgspecv1.Descriptor{
			MediaType: registry.ConfigMediaType,
			Size:      int64(len(configData)),
			Digest:    c.configDigest,
		},
		Layers: []imgspecv1.Descriptor{
			{
				MediaType: c.layerMediaType,
				Size:      layerSize,
				Digest:    layerDigest,
			},
		},
		Annotations: maps.Clone(c.annotations),
	}
	manifestData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	c.manifestDigest = digest.SHA256.FromBytes(manifestData)
	c.layers = append(c.layers, layerDigest)

	// Construct file to containers OCI format
	err = os.Mkdir(filepath.Join(cd, c.manifestDigest.Encoded()), 0755)
	if err != nil {
		return fmt.Errorf("mkdir failed on %q: %w",
			filepath.Join(cd, c.manifestDigest.Encoded()), err)
	}
	// Create share blobs path
	sharedBlobPath := filepath.Join(cd, archive.SharedBlobDir, "sha256")
	if err = os.MkdirAll(sharedBlobPath, 0755); err != nil {
		return fmt.Errorf("mkdir failed %q: %w", sharedBlobPath, err)
	}
	// File layer
	if err = os.Rename(tmpFilePath,
		filepath.Join(sharedBlobPath, layerDigest.Encoded())); err != nil {
		return fmt.Errorf("layer file rename failed: %w", err)
	}
	// Manifest file
	if err = os.WriteFile(filepath.Join(sharedBlobPath, c.manifestDigest.Encoded()), manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	// Config file
	if err = os.WriteFile(filepath.Join(sharedBlobPath, c.configDigest.Encoded()), configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	// OCI Image index
	imagePath := filepath.Join(cd, c.manifestDigest.Encoded())
	if err = os.MkdirAll(filepath.Join(imagePath, imgspecv1.ImageBlobsDir), 0755); err != nil {
		return fmt.Errorf("mkdir failed %q: %w", imagePath, err)
	}
	c.imageDir = imagePath
	index := imgspecv1.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Manifests: []imgspecv1.Descriptor{
			{
				MediaType: imgspecv1.MediaTypeImageManifest,
				Digest:    c.manifestDigest,
				Size:      int64(len(manifestData)),
			},
		},
	}
	indexData, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest index: %w", err)
	}
	if err := os.WriteFile(
		filepath.Join(imagePath, imgspecv1.ImageIndexFile), indexData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest index: %w", err)
	}
	// Image Layout
	layout := imgspecv1.ImageLayout{
		Version: imgspecv1.ImageLayoutVersion,
	}
	layoutData, err := json.Marshal(layout)
	if err != nil {
		return fmt.Errorf("failed to marshal image layout: %w", err)
	}
	if err := os.WriteFile(
		filepath.Join(imagePath, imgspecv1.ImageLayoutFile), layoutData, 0644); err != nil {
		return fmt.Errorf("failed to write layout file: %w", err)
	}
	return nil
}

func (c *common) image() (*archive.Image, error) {
	spec := archive.ImageSpec{
		Arch:        "",
		OS:          "",
		OSVersion:   "",
		OSFeatures:  nil,
		Variant:     "",
		MediaType:   imgspecv1.MediaTypeImageManifest,
		Layers:      nil,
		Config:      c.configDigest,
		Digest:      c.manifestDigest,
		Annotations: c.annotations,
	}
	spec.Layers = slices.Clone(c.layers)
	image := &archive.Image{
		Source:   c.Source(),
		Tag:      c.imageTag,
		ArchList: make([]string, 0),
		OsList:   make([]string, 0),
		Images:   []archive.ImageSpec{spec},
	}
	if image.Tag == "" {
		image.Tag = utils.DefaultTag
	}
	return image, nil
}

// WriteArchive writes chart OCI image into the archive file.
func (c *common) WriteArchive(au *archive.Updater) error {
	if c.cacheDir == "" {
		return fmt.Errorf("chart is not fetched to the cache directory")
	}
	err := au.Append(c.cacheDir)
	if err != nil {
		return fmt.Errorf("write dir %q to archive file: %w",
			c.cacheDir, err)
	}
	index := au.Index()
	image, err := c.image()
	if err != nil {
		return err
	}
	index.Append(image)
	au.SetIndex(index)
	return au.UpdateIndex()
}

func newFileCacheDir() (string, error) {
	cd, err := os.MkdirTemp(utils.HangarCacheDir(), "*")
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	return cd, nil
}
