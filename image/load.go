package image

import (
	"fmt"
	"os"

	"cnrancher.io/image-tools/registry"
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
	sourceImage := fmt.Sprintf("dir:/%s", img.Source)
	destImage := fmt.Sprintf("docker://%s", img.Destination)
	args := []string{"--format=v2s2"}
	if err = registry.SkopeoCopy(sourceImage, destImage, args...); err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	img.Loaded = true
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Debugf("Loaded image %q", destImage)

	return nil
}
