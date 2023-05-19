package mirror

import (
	"fmt"
	"reflect"

	hm "github.com/cnrancher/hangar/pkg/manifest"
	"github.com/cnrancher/hangar/pkg/skopeo"
	u "github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/manifest"
	"github.com/sirupsen/logrus"
)

func (m *Mirror) initLoadDestinationManifest() error {
	// Get destination manifest
	inspectDestImage := fmt.Sprintf("docker://%s:%s", m.Destination, m.Tag)
	out, err := skopeo.Inspect(inspectDestImage, "--raw")
	if err != nil {
		// destination image not found, this error is expected
		return nil
	}

	m.destMIMEType = manifest.GuessMIMEType([]byte(out))
	switch m.destMIMEType {
	case manifest.DockerV2ListMediaType: // schemaVersion 2 manifest.list.v2
		m.destSchema2List, err = manifest.Schema2ListFromManifest([]byte(out))
		if err != nil {
			return fmt.Errorf("initLoadDestinationManifest: %w", err)
		}
	default:
		// ignore other MIME type
	}

	return nil
}

func (m *Mirror) loadedManifestParams() ([]hm.BuildManifestListParam, error) {
	if m.destMIMEType != manifest.DockerV2ListMediaType {
		// if dest manifest does not exists or format is not manifest.list.v2
		return m.SourceManifestSpec(), nil
	}

	srcParams := m.SourceManifestSpec()
	for _, mf := range m.destSchema2List.Manifests {
		dp := hm.BuildManifestListParam{
			Digest: string(mf.Digest),
			Platform: hm.BuildManifestListPlatform{
				Architecture: mf.Platform.Architecture,
				OS:           mf.Platform.OS,
				Variant:      mf.Platform.Variant,
				OsVersion:    mf.Platform.OSVersion,
			},
		}

		var srcContains = false
		for _, sp := range srcParams {
			if reflect.DeepEqual(dp.Platform, sp.Platform) {
				srcContains = true
			}
		}
		if !srcContains {
			srcParams = append(srcParams, dp)
		}
	}

	return srcParams, nil
}

func (m *Mirror) StartLoad() error {
	if m == nil {
		return fmt.Errorf("StartLoad: %w", u.ErrNilPointer)
	}
	if m.Directory == "" {
		return fmt.Errorf("StartLoad: directory is empty string")
	}
	if m.Mode != MODE_LOAD {
		return fmt.Errorf("StartLoad: mirrorer is not in LOAD mode")
	}

	logrus.WithField("M_ID", m.MID).
		Infof("DEST: [%v] TAG: [%v]", m.Destination, m.Tag)

	if err := m.initLoadDestinationManifest(); err != nil {
		return fmt.Errorf("StartLoad: %w", err)
	}

	for _, img := range m.images {
		img.MID = m.MID
		if err := img.Load(); err != nil {
			return fmt.Errorf("StartLoad: %w", err)
		}
	}

	param, err := m.loadedManifestParams()
	if err != nil {
		return fmt.Errorf("StartLoad: %w", err)
	}
	logrus.WithField("M_ID", m.MID).
		Info("creating dest manifest list...")
	if err := m.updateDestManifest(param); err != nil {
		return fmt.Errorf("StartLoad: %w", err)
	}

	logrus.WithField("M_ID", m.MID).
		Infof("loaded \"%s:%s\"", m.Destination, m.Tag)

	return nil
}

func (m *Mirror) Loaded() int {
	var num int = 0
	for _, img := range m.images {
		if img.Loaded {
			num++
		}
	}
	return num
}
