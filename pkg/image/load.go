package image

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cnrancher/hangar/pkg/registry"
	u "github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

func (img *Image) Load() error {
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Debugf("Load image directory: %s", img.Source)
	info, err := os.Stat(img.Source)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("Load: '%s' is not directory", img.Source)
	}
	sourceImage := fmt.Sprintf("oci:/%s", img.Source)
	destImage := fmt.Sprintf("docker://%s", img.Destination)
	share := filepath.Join(img.Directory, "share")
	args := []string{
		"--format=v2s2",
		"--src-shared-blob-dir=" + share,
	}
	if err = registry.SkopeoCopy(sourceImage, destImage, args...); err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	destManifest, err := registry.SkopeoInspect(destImage, "--raw")
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	img.Digest = "sha256:" + u.Sha256Sum(destManifest)
	img.Loaded = true
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Debugf("Loaded image %q", destImage)

	return nil
}
