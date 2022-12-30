package mirror

import (
	"encoding/json"
	"fmt"
	"strings"

	"cnrancher.io/image-tools/image"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
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

	sourceManifestStr string
	destManifestStr   string
	sourceManifest    map[string]interface{}
	destManifest      map[string]interface{}

	images []*image.Image

	// ID of the mirrorer
	MID int

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

	sourceSchemaVersion, err := m.sourceManifestSchemaVersion()
	if err != nil {
		return fmt.Errorf("initImageList: %w", err)
	}
	logrus.WithField("M_ID", m.MID).
		Debugf("sourceSchemaVersion: %v", sourceSchemaVersion)

	switch sourceSchemaVersion {
	case 2:
		sourceMediaType, err := m.sourceManifestMediaType()
		if err != nil {
			return fmt.Errorf("initImageList: %w", err)
		}
		logrus.WithField("M_ID", m.MID).
			Debugf("sourceMediaType: %v", sourceMediaType)
		switch sourceMediaType {
		case u.MediaTypeManifestListV2:
			logrus.WithField("M_ID", m.MID).
				Infof("[%s:%s] is manifest.list.v2", m.Source, m.Tag)
			if err := m.initImageListByListV2(); err != nil {
				return fmt.Errorf("initImageList: %w", err)
			}
		case u.MediaTypeManifestV2:
			logrus.WithField("M_ID", m.MID).
				Infof("[%s:%s] is manifest.v2", m.Source, m.Tag)
			if err := m.initImageListByV2(); err != nil {
				return fmt.Errorf("initImageList: %w", err)
			}
		default:
			return fmt.Errorf("initImageList: %w", u.ErrInvalidMediaType)
		}
	case 1:
		logrus.WithField("M_ID", m.MID).
			Infof("[%s:%s] is manifest.v1", m.Source, m.Tag)
		if err := m.initImageListByV1(); err != nil {
			return fmt.Errorf("initImageList: %w", err)
		}
	default:
		return fmt.Errorf("initImageList: %w", u.ErrInvalidSchemaVersion)
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

// SourceDigests gets the digest list of the copied source image
func (m *Mirror) SourceDigests() []string {
	var digests []string = make([]string, 0)
	for _, img := range m.images {
		if img.Copied {
			digests = append(digests, img.Digest)
		}
	}
	return digests
}

// DestinationDigests gets the exists digest list of the destination image
func (m *Mirror) DestinationDigests() []string {
	var digests []string = make([]string, 0)
	if m.destManifest == nil {
		return digests
	}
	schemaFloat64, ok := m.destManifest["schemaVersion"].(float64)
	if !ok {
		return digests
	}
	schema := int(schemaFloat64)

	switch schema {
	case 1:
		return digests
	case 2:
		mediaType, ok := m.destManifest["mediaType"].(string)
		if !ok {
			return digests
		}
		switch mediaType {
		case u.MediaTypeManifestListV2:
			manifests, ok := m.destManifest["manifests"].([]interface{})
			if !ok {
				return digests
			}
			for _, v := range manifests {
				manifest, ok := v.(map[string]interface{})
				if !ok {
					return digests
				}
				platform, ok := manifest["platform"].(map[string]interface{})
				if !ok {
					return digests
				}
				arch, ok := platform["architecture"].(string)
				if !ok {
					return digests
				}
				if !m.HasArch(arch) {
					continue
				}
				digest, ok := manifest["digest"].(string)
				if !ok {
					return digests
				}
				digests = append(digests, digest)
			}
			return digests
		case u.MediaTypeManifestV2:
			// dest mediaType is not manifest.list, return empty slice
			return digests
		}
	}
	return digests
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

	// Get source manifest list
	inspectSourceImage := fmt.Sprintf("docker://%s:%s", m.Source, m.Tag)
	out, err = registry.SkopeoInspect(inspectSourceImage, "--raw")
	if err != nil {
		return fmt.Errorf("inspect source image failed: %w", err)
	}

	if err = json.NewDecoder(strings.NewReader(out)).
		Decode(&m.sourceManifest); err != nil {
		return fmt.Errorf("decode source manifest json: %w", err)
	}
	m.sourceManifestStr = out

	// Skip inspect destination image if not in mirror mode
	if m.Mode != MODE_MIRROR {
		return nil
	}

	// Get destination manifest list
	inspectDestImage := fmt.Sprintf("docker://%s:%s", m.Destination, m.Tag)
	out, err = registry.SkopeoInspect(inspectDestImage, "--raw")
	if err != nil {
		// destination image not found, this error is expected
		return nil
	}
	m.destManifestStr = out

	if err = json.NewDecoder(strings.NewReader(out)).
		Decode(&m.destManifest); err != nil {
		return fmt.Errorf("decode destination manifest json: %w", err)
	}

	return nil
}

func (m *Mirror) initImageListByListV2() error {
	var (
		manifest  map[string]interface{}
		digest    string
		platform  map[string]interface{}
		arch      string
		variant   string
		osVersion string
		osType    string
		ok        bool
	)

	logrus.WithField("M_ID", m.MID).Debug("Start initImageListByListV2")
	manifests, ok := m.sourceManifest["manifests"].([]interface{})
	if !ok {
		return fmt.Errorf("reading manifests: %w", u.ErrReadJsonFailed)
	}

	var images int = 0
	for _, v := range manifests {
		if manifest, ok = v.(map[string]interface{}); !ok {
			continue
		}
		if digest, ok = manifest["digest"].(string); !ok {
			continue
		}
		logrus.WithField("M_ID", m.MID).Debugf("digest: %s", digest)
		if platform, ok = manifest["platform"].(map[string]interface{}); !ok {
			continue
		}
		if arch, ok = platform["architecture"].(string); !ok {
			continue
		}
		// variant is empty string if not found
		variant, _ = platform["variant"].(string)
		// os.version is only used for windows system
		osVersion, _ = platform["os.version"].(string)
		if !slices.Contains(m.ArchList, arch) {
			logrus.WithField("M_ID", m.MID).
				Debugf("skip copy image %s arch %s", m.Source, arch)
			continue
		}
		logrus.WithField("M_ID", m.MID).Debugf("arch: %s", arch)
		logrus.WithField("M_ID", m.MID).Debugf("variant: %s", variant)
		logrus.WithField("M_ID", m.MID).Debugf("osVersion: %s", osVersion)
		osType, _ = platform["os"].(string)

		sourceImage := fmt.Sprintf("%s@%s", m.Source, digest)
		destImage := fmt.Sprintf("%s:%s",
			m.Destination, image.CopiedTag(m.Tag, osType, arch, variant))
		// create a new image object and append it into image list
		image := image.NewImage(&image.ImageOptions{
			Source:      sourceImage,
			Destination: destImage,
			Tag:         m.Tag,
			Arch:        arch,
			Variant:     variant,
			OS:          osType,
			OsVersion:   osVersion,
			Digest:      digest,
			Directory:   m.Directory,

			SourceSchemaVersion: 2,
			SourceMediaType:     u.MediaTypeManifestListV2,
			MID:                 m.MID,
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
	sourceImage := fmt.Sprintf("docker://%s:%s", m.Source, m.Tag)
	out, err := registry.SkopeoInspect(sourceImage, "--raw", "--config")
	if err != nil {
		return fmt.Errorf("initImageListByV2: %w", err)
	}

	var (
		arch    string
		osType  string
		digest  string
		variant string
		ok      bool
	)

	var sourceOciConfig map[string]interface{}
	json.NewDecoder(strings.NewReader(out)).Decode(&sourceOciConfig)
	if arch, ok = sourceOciConfig["architecture"].(string); !ok {
		return fmt.Errorf("initImageListByV2 read architecture: %w",
			u.ErrReadJsonFailed)
	}
	osType, _ = sourceOciConfig["os"].(string)
	variant, _ = sourceOciConfig["variant"].(string)

	digest = "sha256:" + u.Sha256Sum(m.sourceManifestStr)

	if !slices.Contains(m.ArchList, arch) {
		logrus.WithField("M_ID", m.MID).
			Debugf("skip copy image %s arch %s", m.Source, arch)
		return fmt.Errorf("initImageListByV2: %w", u.ErrNoAvailableImage)
	}

	sourceImage = fmt.Sprintf("%s:%s", m.Source, m.Tag)
	destImage := fmt.Sprintf("%s:%s",
		m.Destination, image.CopiedTag(m.Tag, osType, arch, variant))
	// create a new image object and append it into image list
	image := image.NewImage(&image.ImageOptions{
		Source:              sourceImage,
		Destination:         destImage,
		Tag:                 m.Tag,
		Arch:                arch,
		Variant:             variant,
		OS:                  osType,
		Digest:              digest,
		Directory:           m.Directory,
		SourceSchemaVersion: 2,
		SourceMediaType:     u.MediaTypeManifestV2,
		MID:                 m.MID,
	})
	m.AppendImage(image)

	return nil
}

func (m *Mirror) initImageListByV1() error {
	var (
		arch   string
		ok     bool
		osType string
	)

	sourceImage := fmt.Sprintf("docker://%s:%s", m.Source, m.Tag)
	// `skopeo inspect docker://docker.io/<image>`
	out, err := registry.SkopeoInspect(sourceImage)
	if err != nil {
		return fmt.Errorf("initImageListByV2: %w", err)
	}
	var sourceInfo map[string]interface{}
	json.NewDecoder(strings.NewReader(out)).Decode(&sourceInfo)

	if arch, ok = m.sourceManifest["architecture"].(string); !ok {
		return fmt.Errorf("read architecture failed: %w", u.ErrReadJsonFailed)
	}
	if !slices.Contains(m.ArchList, arch) {
		logrus.WithField("M_ID", m.MID).
			Debugf("skip copy image %s arch %s", m.Source, arch)
	}

	if osType, ok = sourceInfo["Os"].(string); !ok {
		return fmt.Errorf("read Os failed: %w", u.ErrReadJsonFailed)
	}

	// Calculate sha256sum of source manifest
	digest := u.Sha256Sum(m.sourceManifestStr)

	sourceImage = fmt.Sprintf("%s:%s", m.Source, m.Tag)
	// schemaV1 does not have variant
	destImage := fmt.Sprintf("%s:%s",
		m.Destination, image.CopiedTag(m.Tag, osType, arch, ""))
	// create a new image object and append it into image list
	img := image.NewImage(&image.ImageOptions{
		Source:              sourceImage,
		Destination:         destImage,
		Tag:                 m.Tag,
		Arch:                arch,
		Variant:             "",
		OS:                  osType,
		Digest:              digest,
		Directory:           m.Directory,
		SourceSchemaVersion: 1,
		SourceMediaType:     "", // schemaVersion 1 does not have mediaType
		MID:                 m.MID,
	})
	m.AppendImage(img)

	return nil
}

func (m *Mirror) sourceManifestSchemaVersion() (int, error) {
	schemaFloat64, ok := m.sourceManifest["schemaVersion"].(float64)
	if !ok {
		return 0, fmt.Errorf(
			"sourceManifestSchemaVersion read schemaVersion: %w",
			u.ErrReadJsonFailed)
	}
	return int(schemaFloat64), nil
}

func (m *Mirror) sourceManifestMediaType() (string, error) {
	mediaType, ok := m.sourceManifest["mediaType"].(string)
	if !ok {
		return "", fmt.Errorf("SourceManifestMediaType read mediaType: %w",
			u.ErrReadJsonFailed)
	}
	return mediaType, nil
}

func (m *Mirror) compareSourceDestManifest() bool {
	if m.destManifest == nil {
		// dest image does not exist, return false
		logrus.WithField("M_ID", m.MID).
			Debugf("compareSourceDestManifest: dest manifest does not exist")
		return false
	}
	schemaFloat64, ok := m.destManifest["schemaVersion"].(float64)
	if !ok {
		// read json failed, return false
		logrus.WithField("M_ID", m.MID).
			Debugf("compareSourceDestManifest: read schemaVersion failed")
		return false
	}
	var schema int = int(schemaFloat64)
	switch schema {
	// The destination manifest list schemaVersion should be 2
	case 1:
		logrus.WithField("M_ID", m.MID).
			Debugf("compareSourceDestManifest: dest schemaVersion is 1")
		return false
	case 2:
		mediaType, ok := m.destManifest["mediaType"].(string)
		if !ok {
			return false
		}
		switch mediaType {
		case u.MediaTypeManifestListV2:
			// Compare the source image digest list and dest image digest list
			srcDigests := m.SourceDigests()
			dstDigests := m.DestinationDigests()
			logrus.WithField("M_ID", m.MID).
				Debugf("compareSourceDestManifest: ")
			logrus.WithField("M_ID", m.MID).
				Debugf("  srcDigests: %v", srcDigests)
			logrus.WithField("M_ID", m.MID).
				Debugf("  dstDigests: %v", dstDigests)
			return slices.Compare(srcDigests, dstDigests) == 0
		case u.MediaTypeManifestV2:
			// The destination manifest mediaType should be 'manifest.list.v2'
			logrus.WithField("M_ID", m.MID).
				Debugf("compareSourceDestManifest: dest mediaType is m.v2")
			return false
		}
	}

	return false
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
