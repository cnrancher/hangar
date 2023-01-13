package image

import (
	"fmt"
	"path/filepath"

	"github.com/cnrancher/image-tools/pkg/registry"
	u "github.com/cnrancher/image-tools/pkg/utils"
	"github.com/sirupsen/logrus"
)

func (img *Image) Save() error {
	if img.Directory == "" {
		return fmt.Errorf("Save: img.Directory is empty")
	}

	var err error
	var ok bool

	img.SavedFolder = u.Sha256Sum(img.Destination)
	share := filepath.Join(img.Directory, "share")
	img.Directory = filepath.Join(img.Directory, img.SavedFolder)
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Debugf("SavedFolder: sha256sum(%s)", img.Destination)
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Debugf("Save image Directory: %s", img.Directory)

	// Ensure dir empty
	if ok, err = u.IsDirEmpty(img.Directory); !ok {
		if err != nil {
			return fmt.Errorf("Save: check dir is empty: %w", err)
		}
		return fmt.Errorf("Save: %w", u.ErrDirNotEmpty)
	}

	// skopeo copy docker://<source> oci:/<local_dir>
	sourceImage := fmt.Sprintf("docker://%s", img.Source)
	destImageDir := fmt.Sprintf("oci:/%s", img.Directory)

	args := []string{
		"--dest-compress", // compress image in local dir
		"--dest-compress-format=gzip",
		"--dest-compress-level=9",
		"--dest-shared-blob-dir=" + share,
	}
	err = registry.SkopeoCopy(sourceImage, destImageDir, args...)
	if err != nil {
		return fmt.Errorf("Save: skopeo copy :%w", err)
	}
	img.Saved = true
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Infof("Saved image %q", img.Source)

	return nil
}
