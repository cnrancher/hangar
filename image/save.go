package image

import (
	"fmt"
	"path/filepath"
	"strings"

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

	img.directory = filepath.Join(
		img.directory, strings.TrimLeft(img.digest, "sha256:"))
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
