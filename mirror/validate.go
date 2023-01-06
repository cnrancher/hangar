package mirror

import (
	"encoding/json"
	"fmt"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/containers/image/v5/manifest"
	"github.com/sirupsen/logrus"
)

func (m *Mirror) StartValidate() error {
	if m == nil {
		return fmt.Errorf("StartValidate: %w", u.ErrNilPointer)
	}

	if m.Mode != MODE_MIRROR_VALIDATE {
		return fmt.Errorf("StartValidate: mirrorer is not in VALIDATE mode")
	}

	// Init image list from source and destination
	if err := m.initImageList(); err != nil {
		return fmt.Errorf("StartValidate: %w", err)
	}

	if err := m.validateImages(); err != nil {
		return err
	}

	return nil
}

func (m *Mirror) validateImages() error {
	switch m.destMIMEType {
	case manifest.DockerV2ListMediaType:
		if len(m.destSchema2List.Manifests) == 0 {
			return fmt.Errorf("%q: destination manifest list is empty",
				m.Destination)
		}
	case "":
		return fmt.Errorf("%q: destination manifest does not exists",
			m.Destination)
	default:
		return fmt.Errorf("%q: destination manifest MIME type unknow: %v",
			m.Destination, m.destMIMEType)
	}
	switch m.sourceMIMEType {
	case manifest.DockerV2Schema1MediaType,
		manifest.DockerV2Schema1SignedMediaType:
		// source is schemaVersion1
		if len(m.destSchema2List.Manifests) != 1 {
			return fmt.Errorf("destination manifest list length should be 1")
		}
		// do not compare digests since the digest of schemaVersion1 is
		// different with schemaVersion2, compare arch, os, variant,
		// os.version, etc...
		if m.sourceSchema2V1Image.Architecture !=
			m.destSchema2List.Manifests[0].Platform.Architecture {
			return fmt.Errorf("source arch %q != dest arch %q",
				m.sourceSchema2V1Image.Architecture,
				m.destSchema2List.Manifests[0].Platform.Architecture)
		}
		if m.sourceSchema2V1Image.OS !=
			m.destSchema2List.Manifests[0].Platform.OS {
			return fmt.Errorf("source os %q != dest os %q",
				m.sourceSchema2V1Image.OS,
				m.destSchema2List.Manifests[0].Platform.OS)
		}
		if m.sourceSchema2V1Image.Variant !=
			m.destSchema2List.Manifests[0].Platform.Variant {
			return fmt.Errorf("source os %q != dest os %q",
				m.sourceSchema2V1Image.Variant,
				m.destSchema2List.Manifests[0].Platform.Variant)
		}
		if m.destSchema2List.Manifests[0].Platform.OSVersion != "" {
			return fmt.Errorf("dest os.version is %q, should be empty",
				m.destSchema2List.Manifests[0].Platform.OSVersion)
		}
	case manifest.DockerV2Schema2MediaType:
		// source is schemaVersion2 manifest.v2
		if len(m.destSchema2List.Manifests) != 1 {
			return fmt.Errorf("destination manifest list length should be 1")
		}
		// compare digests
		srcDigest := m.images[0].Digest
		dstDigest := m.destSchema2List.Manifests[0].Digest
		if srcDigest != string(dstDigest) {
			return fmt.Errorf("source digest %q != dest digest %q",
				srcDigest, dstDigest)
		}
		// skopeo inspect docker//<dest>@sha256:<dest-digest> --raw
		destImage := fmt.Sprintf("docker://%s@%s", m.Destination, dstDigest)
		_, err := registry.SkopeoInspect(destImage, "--raw")
		if err != nil {
			return fmt.Errorf("failed to inspect dest image %s: %v",
				m.Destination, err)
		}
		// compare image arch, os, variant, etc...
		if m.sourceSchema2V1Image.Architecture !=
			m.destSchema2List.Manifests[0].Platform.Architecture {
			return fmt.Errorf("source arch %q != dest arch %q",
				m.sourceSchema2V1Image.Architecture,
				m.destSchema2List.Manifests[0].Platform.Architecture)
		}
		if m.sourceSchema2V1Image.OS !=
			m.destSchema2List.Manifests[0].Platform.OS {
			return fmt.Errorf("source os %q != dest os %q",
				m.sourceSchema2V1Image.OS,
				m.destSchema2List.Manifests[0].Platform.OS)
		}
		if m.sourceSchema2V1Image.Variant !=
			m.destSchema2List.Manifests[0].Platform.Variant {
			return fmt.Errorf("source os %q != dest os %q",
				m.sourceSchema2V1Image.Variant,
				m.destSchema2List.Manifests[0].Platform.Variant)
		}
		if m.destSchema2List.Manifests[0].Platform.OSVersion != "" {
			return fmt.Errorf("dest os.version is %q, should be empty",
				m.destSchema2List.Manifests[0].Platform.OSVersion)
		}
	case manifest.DockerV2ListMediaType:
		// source is schemaVersion2 manifest.list.v2
		// dest manifest list length should be larger than 0
		// compare images
		for i := range m.images {
			m.images[i].Copied = true
		}
		srcSpecs := m.SourceManifestSpec()
		dstSpecs := m.DestinationManifestSpec()
		if !CompareBuildxManifests(srcSpecs, dstSpecs) {
			srcJson, _ := json.MarshalIndent(srcSpecs, "", "  ")
			dstJson, _ := json.MarshalIndent(dstSpecs, "", "  ")
			logrus.WithField("M_ID", m.MID).
				Errorf("srcSpec: %+v", srcJson)
			logrus.WithField("M_ID", m.MID).
				Errorf("dstSpec: %+v", dstJson)
			return fmt.Errorf("source manifest %q != dest %q",
				m.Source, m.Destination)
		}
		failed := false
		failedImages := make([]string, 0, 4)
		for _, v := range dstSpecs {
			// skopeo inspect docker//<dest>@sha256:<dest-digest> --raw
			destImage := fmt.Sprintf("docker://%s@%s", m.Destination, v.Digest)
			_, err := registry.SkopeoInspect(destImage, "--raw")
			if err != nil {
				logrus.WithField("M_ID", m.MID).
					Errorf("failed to inspect dest image %s: %v",
						m.Destination, err)
				failedImages = append(failedImages, destImage)
				failed = true
			}
		}
		if failed {
			return fmt.Errorf("failed to inspect dest image: %v", failedImages)
		}
	}
	logrus.WithField("M_ID", m.MID).
		Infof("PASS [%s:%s] == [%s:%s]",
			m.Source, m.Tag, m.Destination, m.Tag)

	return nil
}
