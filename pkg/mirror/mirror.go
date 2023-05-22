package mirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/credential"
	hm "github.com/cnrancher/hangar/pkg/manifest"
	"github.com/cnrancher/hangar/pkg/mirror/image"
	"github.com/cnrancher/hangar/pkg/skopeo"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

type Mirror struct {
	Source      string
	Destination string
	Tag         string
	Directory   string
	ArchList    []string
	RepoType    int

	sourceMIMEType    string
	sourceSchema1     *manifest.Schema1
	sourceSchema2     *manifest.Schema2
	sourceImageInfo   *types.ImageInspectInfo
	sourceSchema2List *manifest.Schema2List
	sourceOCIIndex    *imgspecv1.Index
	sourceOCIManifest *imgspecv1.Manifest
	destMIMEType      string
	destSchema2List   *manifest.Schema2List

	images []*image.Image

	// ImageList line
	Line string

	// ID of the mirrorer
	MID  int
	Mode int
}

type MirrorOptions struct {
	Source      string
	Destination string
	Tag         string
	Directory   string
	ArchList    []string
	Line        string
	Mode        int
	ID          int
}

const (
	_ = iota + 0x10
	MODE_MIRROR
	MODE_LOAD
	MODE_SAVE
	MODE_MIRROR_VALIDATE
	MODE_LOAD_VALIDATE
)

func NewMirror(opts *MirrorOptions) *Mirror {
	return &Mirror{
		Source:      opts.Source,
		Destination: opts.Destination,
		Tag:         opts.Tag,
		Directory:   opts.Directory,
		ArchList:    slices.Clone(opts.ArchList),
		Line:        opts.Line,
		Mode:        opts.Mode,
		MID:         opts.ID,
	}
}

func (m *Mirror) Start() error {
	switch m.Mode {
	case MODE_MIRROR:
		return m.StartMirror()
	case MODE_LOAD:
		return m.StartLoad()
	case MODE_SAVE:
		return m.StartSave()
	case MODE_MIRROR_VALIDATE:
		return m.MirrorValidate()
	case MODE_LOAD_VALIDATE:
		return m.LoadValidate()
	}
	return fmt.Errorf("unknow mirror mode")
}

func (m *Mirror) StartMirror() error {
	if m == nil {
		return fmt.Errorf("StartMirror: %w", utils.ErrNilPointer)
	}
	if m.Mode != MODE_MIRROR {
		return fmt.Errorf("StartSave: mirror is not in MIRROR mode")
	}
	logrus.WithField("M_ID", m.MID).Debug("Start Mirror")
	logrus.WithField("M_ID", m.MID).Infof("SOURCE: [%v] DEST: [%v] TAG: [%v]",
		m.Source, m.Destination, m.Tag)

	// Init image list from source and destination
	if err := m.initImageList(); err != nil {
		if errors.Is(err, utils.ErrNoAvailableImage) {
			logrus.WithField("M_ID", m.MID).
				Warnf("%v", err)
			return nil
		}
		return fmt.Errorf("StartMirror: %w", err)
	}
	// Copy images
	for _, img := range m.images {
		if err := img.Copy(); err != nil {
			logrus.WithFields(logrus.Fields{"M_ID": m.MID}).Error(err.Error())
		}
	}
	// If there are some images failed to copy
	if m.ImageNum()-m.Copied() != 0 {
		img := make([]string, 0, 3)
		for i := range m.images {
			if !m.images[i].Copied {
				img = append(img, m.images[i].Source)
			}
		}
		return fmt.Errorf("some images failed to copy: %v", img)
	}

	// If the source manifest list does not equal to the dest manifest list
	if !m.compareSourceDestManifest() {
		logrus.WithField("M_ID", m.MID).
			Info("creating dest manifest list...")
		// Create a new dest manifest list
		if err := m.updateDestManifest(m.SourceManifestSpec()); err != nil {
			return fmt.Errorf("Mirror: %w", err)
		}
	} else {
		logrus.WithField("M_ID", m.MID).
			Info("dest manifest list already exists, no need to recreate")
	}

	logrus.WithField("M_ID", m.MID).
		Infof("MIRROR [%s:%s] => [%s:%s]",
			m.Source, m.Tag, m.Destination, m.Tag)

	return nil
}

func (m *Mirror) initImageList() error {
	var err error
	// Init source and destination manifest
	if err = m.initSourceDestinationManifest(); err != nil {
		return fmt.Errorf("initImageList: %w", err)
	}

	switch m.sourceMIMEType {
	case manifest.DockerV2ListMediaType: // schemaVersion 2 manifest.list.v2
		err = m.initSourceImageListByListV2()
	case manifest.DockerV2Schema2MediaType: // schemaVersion 2 manifest.v2
		err = m.initImageListByV2()
	case manifest.DockerV2Schema1MediaType,
		manifest.DockerV2Schema1SignedMediaType: // schemaVersion 1 manifest.v1
		err = m.initImageListByV1()
	case imgspecv1.MediaTypeImageIndex: // OCI image manifest index (list)
		err = m.initSourceImageListByOCIIndexV1()
	case imgspecv1.MediaTypeImageManifest: // OCI image manifest
		err = m.initImageListByOCIManifestV1()
	default:
		return fmt.Errorf("unsupported MIME type %q", m.sourceMIMEType)
	}
	if err != nil {
		if errors.Is(err, utils.ErrNoAvailableImage) {
			return err
		}
		return fmt.Errorf("initImageList: %w", err)
	}

	return nil
}

func (m *Mirror) HasArch(a string) bool {
	return slices.Contains(m.ArchList, a)
}

func (m *Mirror) ImageNum() int {
	return len(m.images)
}

func (m *Mirror) AppendImage(img *image.Image) {
	if img == nil {
		return
	}
	img.IID = m.ImageNum() + 1
	m.images = append(m.images, img)
}

// SourceManifestSpec gets the source manifest data
func (m *Mirror) SourceManifestSpec() []hm.BuildManifestListParam {
	var spec []hm.BuildManifestListParam = make([]hm.BuildManifestListParam, 0)
	for _, img := range m.images {
		if !img.Copied && !img.Loaded {
			continue
		}
		spec = append(spec, hm.BuildManifestListParam{
			Digest: img.Digest,
			Platform: hm.BuildManifestListPlatform{
				Architecture: img.Arch,
				OS:           img.OS,
				Variant:      img.Variant,
				OsVersion:    img.OsVersion,
			},
		})
	}
	return spec
}

// DestinationManifestSpec gets the dest manifest data
func (m *Mirror) DestinationManifestSpec() []hm.BuildManifestListParam {
	var spec []hm.BuildManifestListParam = make([]hm.BuildManifestListParam, 0)
	switch m.destMIMEType {
	case manifest.DockerV2ListMediaType:
		for _, manifest := range m.destSchema2List.Manifests {
			if !m.HasArch(manifest.Platform.Architecture) {
				continue
			}
			spec = append(spec, hm.BuildManifestListParam{
				Digest: string(manifest.Digest),
				Platform: hm.BuildManifestListPlatform{
					Architecture: manifest.Platform.Architecture,
					OS:           manifest.Platform.OS,
					Variant:      manifest.Platform.Variant,
					OsVersion:    manifest.Platform.OSVersion,
				},
			})
		}
	}
	return spec
}

func (m *Mirror) compareSourceDestManifest() bool {
	switch m.destMIMEType {
	case manifest.DockerV2ListMediaType:
		// Compare the source image manifest list with dest manifest list
		srcSpecs := m.SourceManifestSpec()
		dstSpecs := m.DestinationManifestSpec()
		return hm.CompareBuildManifests(srcSpecs, dstSpecs)
	}

	return false
}

// Copied method gets the number of copied images
func (m *Mirror) Copied() int {
	var num int = 0
	for _, img := range m.images {
		if img.Copied {
			num++
		}
	}
	return num
}

func (m *Mirror) initSourceDestinationManifest() error {
	var err error
	var out string

	// Get source manifest
	inspectSourceImage := fmt.Sprintf("docker://%s:%s", m.Source, m.Tag)
	out, err = skopeo.Inspect(inspectSourceImage, "--raw")
	if err != nil {
		return fmt.Errorf("inspect source image failed: %w", err)
	}

	m.sourceMIMEType = manifest.GuessMIMEType([]byte(out))
	logrus.WithField("M_ID", m.MID).
		Infof("[%s:%s] is [%s]", m.Source, m.Tag, m.sourceMIMEType)
	switch m.sourceMIMEType {
	case manifest.DockerV2Schema1MediaType,
		manifest.DockerV2Schema1SignedMediaType: // schemaVersion 1 manifest.v1
		m.sourceSchema1, err = manifest.Schema1FromManifest([]byte(out))
		if err != nil {
			return err
		}
	case manifest.DockerV2Schema2MediaType: // schemaVersion 2 manifest.v2
		m.sourceSchema2, err = manifest.Schema2FromManifest([]byte(out))
		if err != nil {
			return err
		}
	case manifest.DockerV2ListMediaType: // schemaVersion 2 manifest.list.v2
		m.sourceSchema2List, err = manifest.Schema2ListFromManifest([]byte(out))
		if err != nil {
			return err
		}
	case imgspecv1.MediaTypeImageIndex: // OCI image index (list)
		m.sourceOCIIndex = &imgspecv1.Index{}
		err = json.Unmarshal([]byte(out), m.sourceOCIIndex)
		if err != nil {
			return err
		}
	case imgspecv1.MediaTypeImageManifest: // OCI image manifest
		m.sourceOCIManifest = &imgspecv1.Manifest{}
		err = json.Unmarshal([]byte(out), m.sourceOCIManifest)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported MIME type %q", m.sourceMIMEType)
	}

	// Skip inspect destination image if not in mirror mode
	if m.Mode != MODE_MIRROR && m.Mode != MODE_MIRROR_VALIDATE {
		return nil
	}

	// Get destination manifest
	inspectDestImage := fmt.Sprintf("docker://%s:%s", m.Destination, m.Tag)
	out, err = skopeo.Inspect(inspectDestImage, "--raw")
	if err != nil {
		// destination image not found, this error is expected
		return nil
	}

	m.destMIMEType = manifest.GuessMIMEType([]byte(out))
	switch m.destMIMEType {
	case manifest.DockerV2ListMediaType: // schemaVersion 2 manifest.list.v2
		m.destSchema2List, err = manifest.Schema2ListFromManifest([]byte(out))
		if err != nil {
			return err
		}
	default:
		// ignore other MIME type
	}

	return nil
}

func (m *Mirror) initSourceImageListByListV2() error {
	logrus.WithField("M_ID", m.MID).Debug("Start initImageListByListV2")
	var images int = 0
	for _, manifest := range m.sourceSchema2List.Manifests {
		if !slices.Contains(m.ArchList, manifest.Platform.Architecture) {
			logrus.WithField("M_ID", m.MID).
				Debugf("skip image %s arch %s",
					m.Source, manifest.Platform.Architecture)
			continue
		}
		extra := []string{}
		if manifest.Platform.OSVersion != "" {
			extra = append(extra, manifest.Platform.OSVersion)
		}
		copiedTag := image.CopiedTag(
			m.Tag,
			manifest.Platform.OS,
			manifest.Platform.Architecture,
			manifest.Platform.Variant,
			extra...,
		)
		sourceImage := fmt.Sprintf("%s@%s", m.Source, manifest.Digest)
		destImage := fmt.Sprintf("%s:%s", m.Destination, copiedTag)
		// create a new image object and append it into image list
		image := image.NewImage(&image.ImageOptions{
			Source:      sourceImage,
			Destination: destImage,
			Tag:         m.Tag,
			Arch:        manifest.Platform.Architecture,
			Variant:     manifest.Platform.Variant,
			OS:          manifest.Platform.OS,
			OsVersion:   manifest.Platform.OSVersion,
			Digest:      string(manifest.Digest),
			Directory:   m.Directory,

			SourceMediaType: m.sourceMIMEType,
			MID:             m.MID,
		})
		m.AppendImage(image)
		images++
	}

	if images == 0 {
		logrus.WithField("M_ID", m.MID).Warnf("[%s] does not have arch %v",
			m.Source, m.ArchList)
		return utils.ErrNoAvailableImage
	}

	return nil
}

func (m *Mirror) initSourceImageListByOCIIndexV1() error {
	logrus.WithField("M_ID", m.MID).
		Debug("Start initSourceImageListByOCIIndexV1")
	var images int = 0
	for _, manifest := range m.sourceOCIIndex.Manifests {
		if !slices.Contains(m.ArchList, manifest.Platform.Architecture) {
			logrus.WithField("M_ID", m.MID).
				Debugf("skip image %s arch %s",
					m.Source, manifest.Platform.Architecture)
			continue
		}
		extra := []string{}
		if manifest.Platform.OSVersion != "" {
			extra = append(extra, manifest.Platform.OSVersion)
		}
		copiedTag := image.CopiedTag(
			m.Tag,
			manifest.Platform.OS,
			manifest.Platform.Architecture,
			manifest.Platform.Variant,
			extra...,
		)
		sourceImage := fmt.Sprintf("%s@%s", m.Source, manifest.Digest)
		destImage := fmt.Sprintf("%s:%s", m.Destination, copiedTag)
		// create a new image object and append it into image list
		image := image.NewImage(&image.ImageOptions{
			Source:      sourceImage,
			Destination: destImage,
			Tag:         m.Tag,
			Arch:        manifest.Platform.Architecture,
			Variant:     manifest.Platform.Variant,
			OS:          manifest.Platform.OS,
			OsVersion:   manifest.Platform.OSVersion,
			Digest:      string(manifest.Digest),
			Directory:   m.Directory,

			SourceMediaType: m.sourceMIMEType,
			MID:             m.MID,
		})
		m.AppendImage(image)
		images++
	}

	if images == 0 {
		logrus.WithField("M_ID", m.MID).Warnf("[%s] does not have arch %v",
			m.Source, m.ArchList)
		return utils.ErrNoAvailableImage
	}

	return nil
}

func (m *Mirror) initImageListByV2() error {
	var err error
	m.sourceImageInfo, err = m.sourceSchema2.Inspect(
		func(bi types.BlobInfo) ([]byte, error) {
			// get source image config
			sourceImage := fmt.Sprintf("docker://%s:%s", m.Source, m.Tag)
			o, e := skopeo.Inspect(sourceImage, "--raw", "--config")
			return []byte(o), e
		},
	)
	if err != nil {
		return fmt.Errorf("initImageListByV2: %w", err)
	}

	if !slices.Contains(m.ArchList, m.sourceImageInfo.Architecture) {
		logrus.WithField("M_ID", m.MID).
			Debugf("skip image %s arch %s",
				m.Source, m.sourceImageInfo.Architecture)
		return utils.ErrNoAvailableImage
	}

	copiedTag := image.CopiedTag(
		m.Tag,
		m.sourceImageInfo.Os,
		m.sourceImageInfo.Architecture,
		m.sourceImageInfo.Variant)
	sourceImage := fmt.Sprintf("%s:%s", m.Source, m.Tag)
	destImage := fmt.Sprintf("%s:%s", m.Destination, copiedTag)
	digest, err := skopeo.Inspect(
		"docker://"+sourceImage, "--format", "{{ .Digest }}")
	if err != nil {
		return fmt.Errorf("initImageListByV2: %w", err)
	}
	// create a new image object and append it into image list
	img := image.NewImage(&image.ImageOptions{
		Source:          sourceImage,
		Destination:     destImage,
		Tag:             m.Tag,
		Arch:            m.sourceImageInfo.Architecture,
		Variant:         m.sourceImageInfo.Variant,
		OS:              m.sourceImageInfo.Os,
		Digest:          strings.TrimSpace(digest),
		Directory:       m.Directory,
		SourceMediaType: m.sourceMIMEType,
		MID:             m.MID,
	})
	m.AppendImage(img)

	return nil
}

func (m *Mirror) initImageListByOCIManifestV1() error {
	var err error
	m.sourceImageInfo = &types.ImageInspectInfo{}
	sourceImage := fmt.Sprintf("docker://%s:%s", m.Source, m.Tag)
	o, err := skopeo.Inspect(sourceImage, "--raw", "--config")
	if err != nil {
		return fmt.Errorf("initImageListByOCIManifestV1: %w", err)
	}
	err = json.Unmarshal([]byte(o), m.sourceImageInfo)
	if err != nil {
		return fmt.Errorf("initImageListByOCIManifestV1: %w", err)
	}

	if !slices.Contains(m.ArchList, m.sourceImageInfo.Architecture) {
		logrus.WithField("M_ID", m.MID).
			Debugf("skip image %s arch %s",
				m.Source, m.sourceImageInfo.Architecture)
		return utils.ErrNoAvailableImage
	}

	copiedTag := image.CopiedTag(
		m.Tag,
		m.sourceImageInfo.Os,
		m.sourceImageInfo.Architecture,
		m.sourceImageInfo.Variant)
	sourceImage = fmt.Sprintf("%s:%s", m.Source, m.Tag)
	destImage := fmt.Sprintf("%s:%s", m.Destination, copiedTag)
	digest, err := skopeo.Inspect(
		"docker://"+sourceImage, "--format", "{{ .Digest }}")
	if err != nil {
		return fmt.Errorf("initImageListByOCIManifestV1: %w", err)
	}
	// create a new image object and append it into image list
	img := image.NewImage(&image.ImageOptions{
		Source:          sourceImage,
		Destination:     destImage,
		Tag:             m.Tag,
		Arch:            m.sourceImageInfo.Architecture,
		Variant:         m.sourceImageInfo.Variant,
		OS:              m.sourceImageInfo.Os,
		Digest:          strings.TrimSpace(digest),
		Directory:       m.Directory,
		SourceMediaType: m.sourceMIMEType,
		MID:             m.MID,
	})
	m.AppendImage(img)

	return nil
}

func (m *Mirror) initImageListByV1() error {
	var err error
	m.sourceImageInfo, err = m.sourceSchema1.Inspect(
		func(bi types.BlobInfo) ([]byte, error) {
			// get source image config
			sourceImage := fmt.Sprintf("docker://%s:%s", m.Source, m.Tag)
			o, e := skopeo.Inspect(sourceImage, "--raw", "--config")
			return []byte(o), e
		},
	)
	if err != nil {
		return fmt.Errorf("initImageListByV1: %w", err)
	}

	if !slices.Contains(m.ArchList, m.sourceImageInfo.Architecture) {
		logrus.WithField("M_ID", m.MID).
			Debugf("skip image %s arch %s",
				m.Source, m.sourceImageInfo.Architecture)
		return nil
	}

	// Calculate sha256sum of source manifest
	copiedTag := image.CopiedTag(
		m.Tag, m.sourceImageInfo.Os,
		m.sourceImageInfo.Architecture,
		m.sourceImageInfo.Variant)
	sourceImage := fmt.Sprintf("%s:%s", m.Source, m.Tag)
	destImage := fmt.Sprintf("%s:%s", m.Destination, copiedTag)
	digest, err := skopeo.Inspect(
		"docker://"+sourceImage, "--format", "{{ .Digest }}")
	if err != nil {
		return fmt.Errorf("initImageListByV1: %w", err)
	}
	// create a new image object and append it into image list
	img := image.NewImage(&image.ImageOptions{
		Source:          sourceImage,
		Destination:     destImage,
		Tag:             m.Tag,
		Arch:            m.sourceImageInfo.Architecture,
		Variant:         m.sourceImageInfo.Variant,
		OS:              m.sourceImageInfo.Os,
		Digest:          strings.TrimSpace(digest),
		Directory:       m.Directory,
		SourceMediaType: m.sourceMIMEType,
		MID:             m.MID,
	})
	m.AppendImage(img)

	return nil
}

// updateDestManifest
func (m *Mirror) updateDestManifest(params []hm.BuildManifestListParam) error {
	if len(params) == 0 {
		logrus.Warnf("updateDestManifest: no manifest to build")
		return fmt.Errorf("updateDestManifest: image list is empty")
	}

	if config.GetBool("TESTING") {
		return nil
	}

	uname, passwd, _ := credential.GetRegistryCredential(
		utils.GetRegistryName(m.Destination))
	s2, err := hm.BuildManifestList(m.Destination, uname, passwd, params)
	if err != nil {
		return fmt.Errorf("updateDestManifest: %w", err)
	}
	dt, _ := json.MarshalIndent(s2, "", "  ")
	logrus.WithFields(logrus.Fields{"M_ID": m.MID}).
		Debugf("updateDestManifest: %s", string(dt))

	dst := fmt.Sprintf("%s:%s", m.Destination, m.Tag)
	uname, passwd, _ = credential.GetRegistryCredential(
		utils.GetRegistryName(m.Destination))
	err = hm.PushManifest(dst, uname, passwd, dt)
	if err != nil {
		return fmt.Errorf("updateDestManifest: %w", err)
	}

	return nil
}
