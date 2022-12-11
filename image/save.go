package image

import (
	"fmt"
	"path/filepath"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func (img *Image) Save() error {
	if img.Directory == "" {
		return fmt.Errorf("Save: img.Directory is empty")
	}

	var err error
	var ok bool

	destImage := fmt.Sprintf("%s:%s",
		img.Source, CopiedTag(img.Tag, img.OS, img.Arch, img.Variant))
	img.SavedFolder = u.Sha256Sum(destImage)
	img.Directory = filepath.Join(img.Directory, img.SavedFolder)
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Debugf("SavedFolder: sha256sum(%s)", img.SavedFolder)
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Infof("Save image Directory: %s", img.Directory)

	// Ensure dir empty
	if ok, err = u.IsDirEmpty(img.Directory); !ok {
		if err != nil {
			return fmt.Errorf("Save: check dir is empty: %w", err)
		}
		return fmt.Errorf("Save: %w", u.ErrDirNotEmpty)
	}

	// skopeo copy docker://<source> dir://<local_dir>
	sourceImage := fmt.Sprintf("docker://%s", img.Source)
	destImageDir := fmt.Sprintf("dir:/%s", img.Directory)

	// Convert image manifest schemaVersion to v2, mediaType to 'manifest.v2'
	// when saving image to local dir.
	args := []string{
		"--format=v2s2",
		"--override-arch=" + img.Arch,
		"--dest-compress", // compress image in local dir
		"--dest-compress-format=gzip",
	}
	if img.OS != "" {
		args = append(args, "--override-os="+img.OS)
	}
	if img.Variant != "" {
		args = append(args, "--override-variant="+img.Arch)
	}
	err = registry.SkopeoCopy(sourceImage, destImageDir, args...)
	if err != nil {
		return fmt.Errorf("Save: skopeo copy :%w", err)
	}
	img.Saved = true

	return nil
}
