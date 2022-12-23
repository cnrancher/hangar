package image

import (
	"fmt"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func (img *Image) Copy() error {
	if img == nil {
		return u.ErrNilPointer
	}

	if img.Source == "" || img.Destination == "" || img.Arch == "" {
		return u.ErrInvalidParameter
	}

	if err := img.copyIfChanged(); err != nil {
		return fmt.Errorf("Copy: %w", err)
	}

	if img.SourceSchemaVersion == 1 {
		// get digests from copied dest image
		destImage := fmt.Sprintf("docker://%s", img.Destination)
		// `skopeo inspect docker://docker.io/${repository}:${version}-${arch}`
		out, err := registry.SkopeoInspect(destImage, "--raw")
		if err != nil {
			return fmt.Errorf("Copy: %w", err)
		}
		Digest := "sha256:" + u.Sha256Sum(out)
		img.Digest = Digest
	}

	img.Copied = true
	return nil
}

func (img *Image) copyIfChanged() error {
	var (
		srcDockerImage string
		dstDockerImage string
	)

	// docker://registry/repository:${ORIGINAL_TAG}-${ARCH}${VARIANT}
	srcDockerImage = fmt.Sprintf("docker://%s", img.Source)
	dstDockerImage = fmt.Sprintf("docker://%s", img.Destination)

	destManifest, err := registry.SkopeoInspect(dstDockerImage, "--raw")
	if err != nil {
		// if destination image not found, set destManifest to empty string
		destManifest = ""
	}
	var destDigest string = "NEW_IMAGE"
	if destManifest != "" {
		destDigest = "sha256:" + u.Sha256Sum(destManifest)
	}
	// compare the source manifest with the dest manifest
	if img.Digest == destDigest {
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.MID,
			"IMG_ID": img.IID}).
			Infof("Unchanged: [%s] == [%s]", img.Source, img.Destination)
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.MID,
			"IMG_ID": img.IID}).
			Debugf("Source Digest: %s", img.Digest)
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.MID,
			"IMG_ID": img.IID}).
			Debugf("destin Digest: %s", destDigest)
		return nil
	}
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Infof("Digest changed: [%s] => [%s]", img.Digest, destDigest)
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Infof("Copying: [%s] => [%s]", img.Source, img.Destination)
	args := []string{"--format=v2s2"}
	return registry.SkopeoCopy(srcDockerImage, dstDockerImage, args...)
}
