package mirror

import (
	"encoding/json"
	"fmt"

	"cnrancher.io/image-tools/image"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/containers/image/v5/manifest"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const (
	REPO_TYPE_DEFAULT = iota
	REPO_TYPE_HARBOR_V1
	REPO_TYPE_HARBOR_V2
)

type Mirror struct {
	Source      string
	Destination string
	Tag         string
	Directory   string
	ArchList    []string
	RepoType    int

	sourceManifestStr string // string data for source manifest
	destManifestStr   string // string data for dest manifest

	sourceMIMEType       string
	sourceSchema1        manifest.Schema1
	sourceSchema2        manifest.Schema2
	sourceSchema2V1Image manifest.Schema2V1Image
	sourceSchema2List    manifest.Schema2List
	destMIMEType         string
	destSchema2List      manifest.Schema2List

	images []*image.Image

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
	RepoType    int
	Mode        int
	ID          int
}

const (
	_ = iota + 0x10
	MODE_MIRROR
	MODE_LOAD
	MODE_SAVE
	MODE_VALIDATE
)

func NewMirror(opts *MirrorOptions) *Mirror {
	return &Mirror{
		Source:      opts.Source,
		Destination: opts.Destination,
		Tag:         opts.Tag,
		Directory:   opts.Directory,
		ArchList:    slices.Clone(opts.ArchList),
		RepoType:    opts.RepoType,
		Mode:        opts.Mode,
		MID:         opts.ID,
	}
}

func (m *Mirror) StartMirror() error {
	if m == nil {
		return fmt.Errorf("StartMirror: %w", u.ErrNilPointer)
	}
	if m.Mode != MODE_MIRROR {
		return fmt.Errorf("StartSave: mirror is not in MIRROR mode")
	}
	logrus.WithField("M_ID", m.MID).Debug("Start Mirror")
	logrus.WithField("M_ID", m.MID).Infof("SOURCE: [%v] DEST: [%v] TAG: [%v]",
		m.Source, m.Destination, m.Tag)

	// Init image list from source and destination
	if err := m.initImageList(); err != nil {
		return fmt.Errorf("StartMirror: %w", err)
	}
	// Copy images
	for _, img := range m.images {
		if err := img.Copy(); err != nil {
			logrus.WithFields(logrus.Fields{"M_ID": m.MID}).Error(err.Error())
		}
	}
	// If the source manifest list does not equal to the dest manifest list
	if !m.compareSourceDestManifest() {
		logrus.WithField("M_ID", m.MID).
			Info("Creating dest manifest list...")
		// Create a new dest manifest list
		if err := m.updateDestManifest(); err != nil {
			return fmt.Errorf("Mirror: %w", err)
		}
	} else {
		logrus.WithField("M_ID", m.MID).
			Info("Dest manifest list already exists, no need to recreate")
	}

	logrus.WithField("M_ID", m.MID).
		Infof("Successfully copied %s:%s => %s:%s.",
			m.Source, m.Tag, m.Destination, m.Tag)

	return nil
}

func (m *Mirror) initImageList() error {
	// Init source and destination manifest
	if err := m.initSourceDestinationManifest(); err != nil {
		return fmt.Errorf("initImageList: %w", err)
	}

	switch m.sourceMIMEType {
	case manifest.DockerV2ListMediaType: // schemaVersion 2 manifest.list.v2
		logrus.WithField("M_ID", m.MID).
			Infof("[%s:%s] is manifest.list.v2", m.Source, m.Tag)
		if err := m.initSourceImageListByListV2(); err != nil {
			return fmt.Errorf("initImageList: %w", err)
		}
	case manifest.DockerV2Schema2MediaType: // schemaVersion 2 manifest.v2
		logrus.WithField("M_ID", m.MID).
			Infof("[%s:%s] is manifest.v2", m.Source, m.Tag)
		if err := m.initImageListByV2(); err != nil {
			return fmt.Errorf("initImageList: %w", err)
		}
	case manifest.DockerV2Schema1MediaType,
		manifest.DockerV2Schema1SignedMediaType: // schemaVersion 1 manifest.v1
		logrus.WithField("M_ID", m.MID).
			Infof("[%s:%s] is manifest.v1", m.Source, m.Tag)
		if err := m.initImageListByV1(); err != nil {
			return fmt.Errorf("initImageList: %w", err)
		}
	default:
		return fmt.Errorf("unsupported MIME type %q", m.sourceMIMEType)
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

// SourceManifestSpec gets the source manifest data used by docker-buildx
func (m *Mirror) SourceManifestSpec() []DockerBuildxManifest {
	var spec []DockerBuildxManifest = make([]DockerBuildxManifest, 0)
	for _, img := range m.images {
		if img.Copied {
			spec = append(spec, DockerBuildxManifest{
				Digest: img.Digest,
				Platform: DockerBuildxPlatform{
					Architecture: img.Arch,
					OS:           img.OS,
					Variant:      img.Variant,
					OsVersion:    img.OsVersion,
				},
			})
		}
	}
	return spec
}

// DestinationManifestSpec gets the dest manifest data used by docker-buildx
func (m *Mirror) DestinationManifestSpec() []DockerBuildxManifest {
	var spec []DockerBuildxManifest = make([]DockerBuildxManifest, 0)
	switch m.destMIMEType {
	case manifest.DockerV2ListMediaType:
		for _, manifest := range m.destSchema2List.Manifests {
			if !m.HasArch(manifest.Platform.Architecture) {
				continue
			}
			spec = append(spec, DockerBuildxManifest{
				Digest: string(manifest.Digest),
				Platform: DockerBuildxPlatform{
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
		return CompareBuildxManifests(srcSpecs, dstSpecs)
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
	out, err = registry.SkopeoInspect(inspectSourceImage, "--raw")
	if err != nil {
		return fmt.Errorf("inspect source image failed: %w", err)
	}
	m.sourceManifestStr = out

	m.sourceMIMEType = manifest.GuessMIMEType([]byte(out))
	switch m.sourceMIMEType {
	case manifest.DockerV2Schema1MediaType,
		manifest.DockerV2Schema1SignedMediaType: // schemaVersion 1 manifest.v1
		if err := json.Unmarshal([]byte(out), &m.sourceSchema1); err != nil {
			return err
		}
	case manifest.DockerV2Schema2MediaType: // schemaVersion 2 manifest.v2
		if err := json.Unmarshal([]byte(out), &m.sourceSchema2); err != nil {
			return err
		}
	case manifest.DockerV2ListMediaType: // schemaVersion 2 manifest.list.v2
		err := json.Unmarshal([]byte(out), &m.sourceSchema2List)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported MIME type %q", m.sourceMIMEType)
	}

	// Skip inspect destination image if not in mirror mode
	if m.Mode != MODE_MIRROR {
		return nil
	}

	// Get destination manifest
	inspectDestImage := fmt.Sprintf("docker://%s:%s", m.Destination, m.Tag)
	out, err = registry.SkopeoInspect(inspectDestImage, "--raw")
	if err != nil {
		// destination image not found, this error is expected
		return nil
	}

	m.destMIMEType = manifest.GuessMIMEType([]byte(out))
	switch m.destMIMEType {
	case manifest.DockerV2ListMediaType: // schemaVersion 2 manifest.list.v2
		err := json.Unmarshal([]byte(out), &m.destSchema2List)
		if err != nil {
			return err
		}
	default:
		// ignore other MIME type
	}
	m.destManifestStr = out

	return nil
}

func (m *Mirror) initSourceImageListByListV2() error {
	logrus.WithField("M_ID", m.MID).Debug("Start initImageListByListV2")
	var images int = 0
	for _, manifest := range m.sourceSchema2List.Manifests {
		if !slices.Contains(m.ArchList, manifest.Platform.Architecture) {
			logrus.WithField("M_ID", m.MID).
				Debugf("skip copy image %s arch %s",
					m.Source, manifest.Platform.Architecture)
			continue
		}
		copiedTag := image.CopiedTag(
			m.Tag,
			manifest.Platform.OS,
			manifest.Platform.Architecture,
			manifest.Platform.Variant)
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
		return fmt.Errorf("initImageListByListV2: %w", u.ErrNoAvailableImage)
	}

	return nil
}

func (m *Mirror) initImageListByV2() error {
	// get source image config
	sourceImage := fmt.Sprintf("docker://%s:%s", m.Source, m.Tag)
	out, err := registry.SkopeoInspect(sourceImage, "--raw", "--config")
	if err != nil {
		return fmt.Errorf("initImageListByV2: %w", err)
	}
	if err := json.Unmarshal([]byte(out), &m.sourceSchema2V1Image); err != nil {
		return fmt.Errorf("initImageListByV2: %w", err)
	}

	if !slices.Contains(m.ArchList, m.sourceSchema2V1Image.Architecture) {
		logrus.WithField("M_ID", m.MID).
			Debugf("skip copy image %s arch %s",
				m.Source, m.sourceSchema2V1Image.Architecture)
		return fmt.Errorf("initImageListByV2: %w", u.ErrNoAvailableImage)
	}

	copiedTag := image.CopiedTag(
		m.Tag,
		m.sourceSchema2V1Image.OS,
		m.sourceSchema2V1Image.Architecture,
		m.sourceSchema2V1Image.Variant)
	sourceImage = fmt.Sprintf("%s:%s", m.Source, m.Tag)
	destImage := fmt.Sprintf("%s:%s", m.Destination, copiedTag)
	// create a new image object and append it into image list
	img := image.NewImage(&image.ImageOptions{
		Source:          sourceImage,
		Destination:     destImage,
		Tag:             m.Tag,
		Arch:            m.sourceSchema2V1Image.Architecture,
		Variant:         m.sourceSchema2V1Image.Variant,
		OS:              m.sourceSchema2V1Image.OS,
		Digest:          "sha256:" + u.Sha256Sum(m.sourceManifestStr),
		Directory:       m.Directory,
		SourceMediaType: m.sourceMIMEType,
		MID:             m.MID,
	})
	m.AppendImage(img)

	return nil
}

func (m *Mirror) initImageListByV1() error {
	// inspect source image config
	sourceImage := fmt.Sprintf("docker://%s:%s", m.Source, m.Tag)
	out, err := registry.SkopeoInspect(sourceImage, "--config")
	if err != nil {
		return fmt.Errorf("initImageListByV1: %w", err)
	}
	err = json.Unmarshal([]byte(out), &m.sourceSchema2V1Image)
	if err != nil {
		return fmt.Errorf("initImageListByV1: %w", err)
	}

	if !slices.Contains(m.ArchList, m.sourceSchema2V1Image.Architecture) {
		logrus.WithField("M_ID", m.MID).
			Debugf("skip copy image %s arch %s",
				m.Source, m.sourceSchema2V1Image.Architecture)
		return nil
	}

	// Calculate sha256sum of source manifest
	digest := "sha267:" + u.Sha256Sum(m.sourceManifestStr)
	copiedTag := image.CopiedTag(
		m.Tag, m.sourceSchema2V1Image.OS,
		m.sourceSchema2V1Image.Architecture,
		m.sourceSchema2V1Image.Variant)
	sourceImage = fmt.Sprintf("%s:%s", m.Source, m.Tag)
	destImage := fmt.Sprintf("%s:%s", m.Destination, copiedTag)
	// create a new image object and append it into image list
	img := image.NewImage(&image.ImageOptions{
		Source:          sourceImage,
		Destination:     destImage,
		Tag:             m.Tag,
		Arch:            m.sourceSchema2V1Image.Architecture,
		Variant:         m.sourceSchema2V1Image.Variant,
		OS:              m.sourceSchema2V1Image.OS,
		Digest:          digest,
		Directory:       m.Directory,
		SourceMediaType: m.sourceMIMEType,
		MID:             m.MID,
	})
	m.AppendImage(img)

	return nil
}

// updateDestManifest
func (m *Mirror) updateDestManifest() error {
	var args []string = []string{
		"imagetools",
		"create",
		fmt.Sprintf("--tag=%s:%s", m.Destination, m.Tag),
	}

	for _, img := range m.images {
		if !img.Copied && !img.Loaded {
			continue
		}
		// args = append(args, img.Destination)
		manifest := DockerBuildxManifest{
			Digest: img.Digest,
			Platform: DockerBuildxPlatform{
				Architecture: img.Arch,
				OS:           img.OS,
				Variant:      img.Variant,
				OsVersion:    img.OsVersion,
			},
		}
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			logrus.Warnf("updateDestManifest: %v", err)
			continue
		}
		logrus.WithFields(logrus.Fields{
			"M_ID":   img.MID,
			"IMG_ID": img.IID}).
			Debugf("updateDestManifest: %s", string(data))
		args = append(args, string(data))
	}

	// docker buildx imagetools create --tag=registry/repository:tag <images>
	if err := registry.DockerBuildx(args...); err != nil {
		return fmt.Errorf("updateDestManifest: %w", err)
	}
	return nil
}
