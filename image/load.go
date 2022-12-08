package image

import (
	"fmt"
	"os"

	"cnrancher.io/image-tools/registry"
	"github.com/sirupsen/logrus"
)

func (img *Image) Load() error {
	if img.directory == "" {
		return fmt.Errorf("Load: directory is empty")
	}
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.mID,
		"IMG_ID": img.iID}).
		Infof("Load image directory: %s", img.directory)
	info, err := os.Stat(img.directory)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("Load: '%s' is not directory", img.directory)
	}

	sourceImage := fmt.Sprintf("dir:/%s", img.source)
	destImage := fmt.Sprintf("docker://%s", img.destination)
	args := []string{"--format=v2s2", "--override-arch=" + img.arch}
	if img.os != "" {
		args = append(args, "--override-os="+img.os)
	}
	if img.variant != "" {
		args = append(args, "--override-variant="+img.arch)
	}
	if err = registry.SkopeoCopy(sourceImage, destImage, args...); err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	img.loaded = true

	return nil
}

func (img *Image) Loaded() bool {
	return img.loaded
}
