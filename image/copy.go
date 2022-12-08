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

	if img.source == "" || img.destination == "" || img.arch == "" {
		return u.ErrInvalidParameter
	}

	if err := img.copyIfChanged(); err != nil {
		return fmt.Errorf("Copy: %w", err)
	}

	if img.sourceSchemaVersion == 1 {
		// get digests from copied dest image
		destImage := fmt.Sprintf("docker://%s", img.destination)
		// `skopeo inspect docker://docker.io/${repository}:${version}-${arch}`
		out, err := registry.SkopeoInspect(destImage, "--raw")
		if err != nil {
			return fmt.Errorf("Copy: %w", err)
		}
		digest := "sha256:" + u.Sha256Sum(out)
		img.digest = digest
	}

	img.copied = true
	return nil
}

func (img *Image) Copied() bool {
	return img.copied
}

func (img *Image) copyIfChanged() error {
	var (
		srcDockerImage string
		dstDockerImage string
	)

	// docker://registry/repository:${ORIGINAL_TAG}-${ARCH}${VARIANT}
	srcDockerImage = fmt.Sprintf("docker://%s", img.source)
	dstDockerImage = fmt.Sprintf("docker://%s", img.destination)

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
	if img.digest == destDigest {
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.mID,
			"IMG_ID": img.iID}).
			Infof("Unchanged: [%s] == [%s]", img.source, img.destination)
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.mID,
			"IMG_ID": img.iID}).
			Infof("source digest: %s", img.digest)
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.mID,
			"IMG_ID": img.iID}).
			Infof("destin digest: %s", destDigest)
		return nil
	}
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.mID,
		"IMG_ID": img.iID}).
		Infof("Digest changed: [%s] => [%s]", img.digest, destDigest)
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.mID,
		"IMG_ID": img.iID}).
		Infof("Copying: [%s] => [%s]", img.source, img.destination)
	args := []string{"--format=v2s2", "--override-arch=" + img.arch}
	if img.os != "" {
		args = append(args, "--override-os="+img.os)
	}
	if img.variant != "" {
		args = append(args, "--override-variant="+img.arch)
	}
	return registry.SkopeoCopy(srcDockerImage, dstDockerImage, args...)
}
