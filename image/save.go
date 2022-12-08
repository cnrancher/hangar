package image

import (
	"fmt"
	"path/filepath"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func (img *Image) Save() error {
	if img.directory == "" {
		return fmt.Errorf("Save: img.directory is empty")
	}

	var err error
	var ok bool

	img.directory = filepath.Join(img.directory, img.digest)
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.mID,
		"IMG_ID": img.iID}).
		Infof("Save image directory: %s", img.directory)

	// Ensure dir exists
	if err = u.EnsureDirExists(img.directory); err != nil {
		return fmt.Errorf("Save: %w", err)
	}
	// Ensure dir empty
	if ok, err = u.IsDirEmpty(img.directory); !ok {
		if err != nil {
			return fmt.Errorf("Save: check dir is empty: %w", err)
		}
		return fmt.Errorf("Save: %w", u.ErrDirNotEmpty)
	}

	// var sourceImage string
	// switch img.sourceSchemaVersion {
	// case 1:
	// 	sourceImage = fmt.Sprintf("docker://%s:%s", img.source, img.tag)
	// case 2:
	// 	switch img.sourceMediaType {
	// 	case u.MediaTypeManifestListV2:
	// 		// registry/repository@sha256:abcdef...
	// 		sourceImage = fmt.Sprintf("docker://%s@%s", img.source, img.digest)
	// 	case u.MediaTypeManifestV2:
	// 		// registry/repository:va.b.c
	// 		sourceImage = fmt.Sprintf("docker://%s:%s", img.source, img.tag)
	// 	default:
	// 		return fmt.Errorf("Save: %w", u.ErrInvalidMediaType)
	// 	}
	// default:
	// 	return fmt.Errorf("Save: %w", u.ErrInvalidSchemaVersion)
	// }

	// skopeo copy docker://<source> dir://<local_dir>
	sourceImage := fmt.Sprintf("docker://%s", img.source)
	destImage := fmt.Sprintf("dir:/%s", img.directory)

	// Convert image manifest schemaVersion to v2, mediaType to 'manifest.v2'
	// when saving image to local dir.
	args := []string{"--format=v2s2", "--override-arch=" + img.arch}
	if img.os != "" {
		args = append(args, "--override-os="+img.os)
	}
	if img.variant != "" {
		args = append(args, "--override-variant="+img.arch)
	}
	err = registry.SkopeoCopy(sourceImage, destImage, args...)
	if err != nil {
		return fmt.Errorf("Save: skopeo copy :%w", err)
	}
	img.saved = true

	return nil
}

func (img *Image) Saved() bool {
	return img.saved
}
