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

// Mirrorer interface is for mirror the images from source registry to
// destination registry
type Mirrorer interface {
	// StartMirror mirrors images from source registry to destination registry
	StartMirror() error

	// StartSave saves images into local directory
	StartSave() error

	// StartLoad loads the images from local directory to dest repository
	StartLoad() error

	// Source gets the source image
	// [registry.io/library/name]
	Source() string

	// Destination gets the destination image
	// [registry.io/library/name]
	Destination() string

	// Tag gets the image tag
	Tag() string

	// Directory gets the directory to save/load
	Directory() string

	// HasArch checks the arch of this image should be copied or not
	HasArch(string) bool

	// ImageNum gets the number of the images
	ImageNum() int

	// AppendImage adds an image to the Mirrorer
	AppendImage(image.Imager)

	// SourceDigests gets the digest list of the copied source image
	SourceDigests() []string

	// DestinationDigests gets the exists digest list of the destination image
	DestinationDigests() []string

	// GetSavedImageTemplate converts this mirrorer to *SavedMirrorTemplate
	GetSavedImageTemplate() *SavedMirrorTemplate

	// Copied method gets the number of copied images
	Copied() int

	// Saved
	Saved() int

	// Load
	Loaded() int

	// Set ID of the Mirrorer
	SetID(string)

	// ID gets the ID of the Mirrorer
	ID() string

	Mode() int
}

type Mirror struct {
	source            string
	destination       string
	tag               string
	directory         string
	availableArchList []string

	sourceManifestStr string
	destManifestStr   string
	sourceManifest    map[string]interface{}
	destManifest      map[string]interface{}

	images []image.Imager

	// ID of the mirrorer
	mID string

	mode int
}

type MirrorOptions struct {
	Source      string
	Destination string
	Tag         string
	Directory   string
	ArchList    []string
	Mode        int
}

const (
	_ = iota + 0x10
	MODE_MIRROR
	MODE_LOAD
	MODE_SAVE
)

func NewMirror(opts *MirrorOptions) *Mirror {
	return &Mirror{
		source:            opts.Source,
		destination:       opts.Destination,
		tag:               opts.Tag,
		directory:         opts.Directory,
		availableArchList: slices.Clone(opts.ArchList),
		mode:              opts.Mode,
	}
}

func (m *Mirror) StartMirror() error {
	if m == nil {
		return fmt.Errorf("StartMirror: %w", u.ErrNilPointer)
	}
	if m.mode != MODE_MIRROR {
		return fmt.Errorf("StartSave: mirrorer is not in MIRROR mode")
	}
	logrus.WithField("M_ID", m.mID).Debug("Start Mirror")

	if err := m.initImageList(); err != nil {
		return fmt.Errorf("StartMirror: %w", err)
	}

	for _, img := range m.images {
		if err := img.Copy(); err != nil {
			logrus.WithFields(logrus.Fields{"M_ID": m.mID}).Error(err.Error())
		}
	}

	// If the source manifest list does not equal to the dest manifest list
	if !m.compareSourceDestManifest() {
		logrus.WithField("M_ID", m.mID).
			Info("Creating dest manifest list...")
		if err := m.updateDestManifest(); err != nil {
			return fmt.Errorf("Mirror: %w", err)
		}
	} else {
		logrus.WithField("M_ID", m.mID).
			Info("Dest manifest list already exists, no need to recreate")
	}

	logrus.WithField("M_ID", m.mID).
		Infof("Successfully copied %s:%s => %s:%s.",
			m.source, m.tag, m.destination, m.tag)

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
	logrus.WithField("M_ID", m.mID).
		Debugf("sourceSchemaVersion: %v", sourceSchemaVersion)

	switch sourceSchemaVersion {
	case 2:
		sourceMediaType, err := m.sourceManifestMediaType()
		if err != nil {
			return fmt.Errorf("initImageList: %w", err)
		}
		logrus.WithField("M_ID", m.mID).
			Debugf("sourceMediaType: %v", sourceMediaType)
		switch sourceMediaType {
		case u.MediaTypeManifestListV2:
			logrus.WithField("M_ID", m.mID).
				Infof("[%s:%s] is manifest.list.v2", m.source, m.tag)
			if err := m.initImageListByListV2(); err != nil {
				return fmt.Errorf("initImageList: %w", err)
			}
		case u.MediaTypeManifestV2:
			logrus.WithField("M_ID", m.mID).
				Infof("[%s:%s] is manifest.v2", m.source, m.tag)
			if err := m.initImageListByV2(); err != nil {
				return fmt.Errorf("initImageList: %w", err)
			}
		default:
			return fmt.Errorf("initImageList: %w", u.ErrInvalidMediaType)
		}
	case 1:
		logrus.WithField("M_ID", m.mID).
			Infof("[%s:%s] is manifest.v1", m.source, m.tag)
		if err := m.initImageListByV1(); err != nil {
			return fmt.Errorf("initImageList: %w", err)
		}
	default:
		return fmt.Errorf("initImageList: %w", u.ErrInvalidSchemaVersion)
	}

	return nil
}

func (m *Mirror) Source() string {
	return m.source
}

func (m *Mirror) Destination() string {
	return m.destination
}

func (m *Mirror) Tag() string {
	return m.tag
}

func (m *Mirror) Directory() string {
	return m.directory
}

func (m *Mirror) HasArch(a string) bool {
	return slices.Contains(m.availableArchList, a)
}

func (m *Mirror) ImageNum() int {
	return len(m.images)
}

func (m *Mirror) AppendImage(img image.Imager) {
	img.SetID(fmt.Sprintf("%02d", m.ImageNum()+1))
	m.images = append(m.images, img)
}

// SourceDigests gets the digest list of the copied source image
func (m *Mirror) SourceDigests() []string {
	var digests []string = make([]string, 0)
	for _, img := range m.images {
		if img.Copied() {
			digests = append(digests, img.Digest())
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
		if img.Copied() {
			num++
		}
	}
	return num
}

func (m *Mirror) SetID(id string) {
	m.mID = id
}

func (m *Mirror) ID() string {
	return m.mID
}

func (m *Mirror) Mode() int {
	return m.mode
}

func (m *Mirror) initSourceDestinationManifest() error {
	var err error
	var out string

	// Get source manifest list
	inspectSourceImage := fmt.Sprintf("docker://%s:%s", m.source, m.tag)
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
	if m.mode != MODE_MIRROR {
		return nil
	}

	// Get destination manifest list
	inspectDestImage := fmt.Sprintf("docker://%s:%s", m.destination, m.tag)
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
		manifest map[string]interface{}
		digest   string
		platform map[string]interface{}
		arch     string
		variant  string
		osType   string
		ok       bool
	)

	logrus.WithField("M_ID", m.mID).Debug("Start initImageListByListV2")
	manifests, ok := m.sourceManifest["manifests"].([]interface{})
	if !ok {
		return fmt.Errorf("reading manifests: %w", u.ErrReadJsonFailed)
	}

	// var images int = 0
	for _, v := range manifests {
		if manifest, ok = v.(map[string]interface{}); !ok {
			continue
		}
		if digest, ok = manifest["digest"].(string); !ok {
			continue
		}
		logrus.WithField("M_ID", m.mID).Debugf("digest: %s", digest)
		if platform, ok = manifest["platform"].(map[string]interface{}); !ok {
			continue
		}
		if arch, ok = platform["architecture"].(string); !ok {
			continue
		}
		// variant is empty string if not found
		variant, _ = platform["variant"].(string)
		if !slices.Contains(m.availableArchList, arch) {
			logrus.WithField("M_ID", m.mID).
				Debugf("skip copy image %s arch %s", m.source, arch)
			continue
		}
		logrus.WithField("M_ID", m.mID).Debugf("arch: %s", arch)
		osType, _ = platform["os"].(string)

		sourceImage := fmt.Sprintf("%s@%s", m.source, digest)
		destImage := fmt.Sprintf("%s:%s",
			m.destination, image.CopiedTag(m.tag, arch, variant))
		// create a new image object and append it into image list
		image := image.NewImage(&image.ImageOptions{
			Source:      sourceImage,
			Destination: destImage,
			Tag:         m.tag,
			Arch:        arch,
			Variant:     variant,
			OS:          osType,
			Digest:      digest,
			Directory:   m.directory,

			SourceSchemaVersion: 2,
			SourceMediaType:     u.MediaTypeManifestListV2,
			MID:                 m.mID,
		})
		m.AppendImage(image)
		// images++
	}

	// if images == 0 {
	// 	logrus.WithField("M_ID", m.mID).Debug("image [%s] does not have arch %v",
	// 		m.source, m.availableArchList)
	// }

	return nil
}

func (m *Mirror) initImageListByV2() error {
	sourceImage := fmt.Sprintf("docker://%s:%s", m.source, m.tag)
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

	if !slices.Contains(m.availableArchList, arch) {
		logrus.WithField("M_ID", m.mID).
			Debugf("skip copy image %s arch %s", m.source, arch)
	}

	sourceImage = fmt.Sprintf("%s:%s", m.source, m.tag)
	destImage := fmt.Sprintf("%s:%s",
		m.destination, image.CopiedTag(m.tag, arch, variant))
	// create a new image object and append it into image list
	image := image.NewImage(&image.ImageOptions{
		Source:              sourceImage,
		Destination:         destImage,
		Tag:                 m.tag,
		Arch:                arch,
		Variant:             variant,
		OS:                  osType,
		Digest:              digest,
		Directory:           m.directory,
		SourceSchemaVersion: 2,
		SourceMediaType:     u.MediaTypeManifestV2,
		MID:                 m.mID,
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

	sourceImage := fmt.Sprintf("docker://%s:%s", m.source, m.tag)
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
	if !slices.Contains(m.availableArchList, arch) {
		logrus.WithField("M_ID", m.mID).
			Debugf("skip copy image %s arch %s", m.source, arch)
	}

	if osType, ok = sourceInfo["Os"].(string); !ok {
		return fmt.Errorf("read Os failed: %w", u.ErrReadJsonFailed)
	}

	// Calculate sha256sum of source manifest
	digest := u.Sha256Sum(m.sourceManifestStr)

	sourceImage = fmt.Sprintf("%s:%s", m.source, m.tag)
	// schemaV1 does not have variant
	destImage := fmt.Sprintf("%s:%s",
		m.destination, image.CopiedTag(m.tag, arch, ""))
	// create a new image object and append it into image list
	img := image.NewImage(&image.ImageOptions{
		Source:              sourceImage,
		Destination:         destImage,
		Tag:                 m.tag,
		Arch:                arch,
		Variant:             "",
		OS:                  osType,
		Digest:              digest,
		Directory:           m.directory,
		SourceSchemaVersion: 1,
		SourceMediaType:     "", // schemaVersion 1 does not have mediaType
		MID:                 m.mID,
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
		logrus.WithField("M_ID", m.mID).
			Infof("compareSourceDestManifest: dest manifest does not exist")
		return false
	}
	schemaFloat64, ok := m.destManifest["schemaVersion"].(float64)
	if !ok {
		// read json failed, return false
		logrus.WithField("M_ID", m.mID).
			Infof("compareSourceDestManifest: read schemaVersion failed")
		return false
	}
	var schema int = int(schemaFloat64)
	switch schema {
	// The destination manifest list schemaVersion should be 2
	case 1:
		logrus.WithField("M_ID", m.mID).
			Infof("compareSourceDestManifest: dest schemaVersion is 1")
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
			logrus.WithField("M_ID", m.mID).
				Infof("compareSourceDestManifest: ")
			logrus.WithField("M_ID", m.mID).
				Infof("  srcDigests: %v", srcDigests)
			logrus.WithField("M_ID", m.mID).
				Infof("  dstDigests: %v", dstDigests)
			return slices.Compare(srcDigests, dstDigests) == 0
		case u.MediaTypeManifestV2:
			// The destination manifest mediaType should be 'manifest.list.v2'
			logrus.WithField("M_ID", m.mID).
				Infof("compareSourceDestManifest: dest mediaType is m.v2")
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
		fmt.Sprintf("--tag=%s:%s", m.destination, m.tag),
	}

	for _, img := range m.images {
		if !img.Copied() && !img.Loaded() {
			continue
		}
		args = append(args, img.Destination())
	}

	// docker buildx imagetools create --tag=registry/repository:tag <images>
	if err := registry.DockerBuildx(args...); err != nil {
		return fmt.Errorf("updateDestManifest: %w", err)
	}
	return nil
}

// ConstructRegistry will re-construct the image url:
//
// If `registryOverride` is empty string, example:
// nginx --> docker.io/nginx (add docker.io prefix)
// reg.io/nginx --> reg.io/nginx (nothing changed)
// reg.io/user/nginx --> reg.io/user/nginx (nothing changed)
//
// If `registryOverride` set, example:
// nginx --> ${registryOverride}/nginx (add ${registryOverride} prefix)
// reg.io/nginx --> ${registryOverride}/nginx (set registry ${registryOverride})
// reg.io/user/nginx --> ${registryOverride}/user/nginx (same as above)
func ConstructRegistry(image, registryOverride string) string {
	s := strings.Split(image, "/")
	if strings.ContainsAny(s[0], ".:") || s[0] == "localhost" {
		if registryOverride != "" {
			s[0] = registryOverride
		}
	} else {
		if registryOverride != "" {
			s = append([]string{registryOverride}, s...)
		} else {
			s = append([]string{u.DockerHubRegistry}, s...)
		}
	}

	return strings.Join(s, "/")
}
