package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cnrancher/hangar/pkg/skopeo"
	"github.com/sirupsen/logrus"
)

func (img *Image) Load() error {
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Debugf("Load image directory: %s", img.Source)
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Infof("loading to [%s]", img.Destination)
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
	if err = skopeo.Copy(sourceImage, destImage, args...); err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	destDigest, err := skopeo.Inspect(destImage, "--format", "{{ .Digest }}")
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	img.Digest = strings.TrimSpace(destDigest)
	img.Loaded = true
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Debugf("Loaded image %q", destImage)

	return nil
}
