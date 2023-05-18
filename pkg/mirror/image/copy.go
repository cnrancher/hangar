package image

import (
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/skopeo"
	u "github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/manifest"
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

	if img.SourceMediaType == manifest.DockerV2Schema1MediaType ||
		img.SourceMediaType == manifest.DockerV2Schema1SignedMediaType {
		// get digests from copied dest image
		destImage := fmt.Sprintf("docker://%s", img.Destination)
		// skopeo inspect docker://docker.io/${repository}:${version}-${arch}
		out, err := skopeo.Inspect(destImage, "--format", "{{ .Digest }}")
		if err != nil {
			return fmt.Errorf("Copy: %w", err)
		}
		Digest := strings.TrimSpace(out)
		img.Digest = Digest
	}

	img.Copied = true
	return nil
}

func (img *Image) copyIfChanged() error {
	// docker://registry/repository:${ORIGINAL_TAG}-${ARCH}${VARIANT}
	srcDockerImage := fmt.Sprintf("docker://%s", img.Source)
	dstDockerImage := fmt.Sprintf("docker://%s", img.Destination)

	if img.SourceMediaType == manifest.DockerV2Schema1MediaType ||
		img.SourceMediaType == manifest.DockerV2Schema1SignedMediaType {
		return skopeo.Copy(
			srcDockerImage, dstDockerImage, "--format=v2s2")
	}

	destDigest, err := skopeo.Inspect(
		dstDockerImage, "--format", "{{ .Digest }}")
	if err != nil {
		destDigest = "NEW_IMAGE"
	}
	destDigest = strings.TrimSpace(destDigest)
	// compare the source manifest with the dest manifest
	if img.Digest == destDigest {
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.MID,
			"IMG_ID": img.IID}).
			Infof("unchanged: [%s] == [%s]", img.Source, img.Destination)
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.MID,
			"IMG_ID": img.IID}).
			Debugf("source digest: %s", img.Digest)
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.MID,
			"IMG_ID": img.IID}).
			Debugf("destin digest: %s", destDigest)
		return nil
	}
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Infof("digest changed: [%s] => [%s]", img.Digest, destDigest)
	logrus.WithFields(logrus.Fields{
		"M_ID":   img.MID,
		"IMG_ID": img.IID}).
		Infof("copying: [%s] => [%s]", img.Source, img.Destination)

	return skopeo.Copy(srcDockerImage, dstDockerImage)
}
