package image

import (
	"fmt"
	"path/filepath"

	"github.com/cnrancher/hangar/pkg/skopeo"
	u "github.com/cnrancher/hangar/pkg/utils"
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
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.MID,
			"IMG_ID": img.IID}).
			Warnf("%q is not empty, delete", img.Directory)
		if err := u.DeleteIfExist(img.Directory); err != nil {
			return fmt.Errorf("Save: %w", err)
		}
	}

	// skopeo copy docker://<source> oci:/<local_dir>
	sourceImage := fmt.Sprintf("docker://%s", img.Source)
	destImageDir := fmt.Sprintf("oci:/%s", img.Directory)

	args := []string{
		"--dest-compress-format=gzip",
		"--dest-compress-level=9",
		"--dest-shared-blob-dir=" + share,
	}
	err = skopeo.Copy(sourceImage, destImageDir, args...)
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
