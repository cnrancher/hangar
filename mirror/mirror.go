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

// Mirrorer interface for mirror the images from source registry to
// destination registry
type Mirrorer interface {
	// Mirror mirrors the image from source registry to destination registry
	Mirror() error

	// Source gets the source image
	// [registry.io/library/name]
	Source() string

	// Destination gets the destination image
	// [registry.io/library/name]
	Destination() string

	// Tag gets the image tag
	Tag() string

	// HasArch checks the arch of this image should be copied or not
	HasArch(string) bool

	// ImageNum gets the number of the images
	ImageNum() int

	// AppendImage adds an image to the Mirrorer
	AppendImage(image.Imagerer)

	// SourceDigests gets the digest list of the copied source image
	SourceDigests() []string

	// DestinationDigests gets the exists digest list of the destination image
	DestinationDigests() []string

	// Copied method gets the number of copied images
	Copied() int

	// Failed method gets the number of images copy failed
	Failed() int

	// Set ID of the Mirrorer
	SetID(string)

	// ID gets the ID of the Mirrorer
	ID() string
}

type Mirror struct {
	source            string
	destination       string
	tag               string
	availableArchList []string

	sourceManifest map[string]interface{}
	destManifest   map[string]interface{}

	images []image.Imagerer

	// ID of the mirrorer
	mID string
}

type MirrorOptions struct {
	Source      string
	Destination string
	Tag         string
	ArchList    []string
}

func NewMirror(opts *MirrorOptions) *Mirror {
	return &Mirror{
		source:            opts.Source,
		destination:       opts.Destination,
		tag:               opts.Tag,
		availableArchList: slices.Clone(opts.ArchList),
	}
}

func (m *Mirror) Mirror() error {
	if m == nil {
		return fmt.Errorf("Mirror: %w", u.ErrNilPointer)
	}

	logrus.WithField("MID", m.mID).Debug("start Mirror")
	// Init source and destination manifest
	if err := m.initSourceDestinationManifest(); err != nil {
		return fmt.Errorf("Mirror: %w", err)
	}

	sourceSchemaVersion, err := m.sourceManifestSchemaVersion()
	if err != nil {
		return fmt.Errorf("Mirror: %w", err)
	}
	logrus.WithField("MID", m.mID).
		Debugf("sourceSchemaVersion: %v", sourceSchemaVersion)

	switch sourceSchemaVersion {
	case 2:
		sourceMediaType, err := m.sourceManifestMediaType()
		if err != nil {
			return fmt.Errorf("Mirror: %w", err)
		}
		logrus.WithField("MID", m.mID).
			Debugf("sourceMediaType: %v", sourceMediaType)
		switch sourceMediaType {
		case u.MediaTypeManifestListV2:
			logrus.WithField("MID", m.mID).
				Infof("[%s:%s] is manifest.list.v2", m.source, m.tag)
			if err := m.initImageListByListV2(); err != nil {
				return fmt.Errorf("Mirror: %w", err)
			}
		case u.MediaTypeManifestV2:
			logrus.WithField("MID", m.mID).
				Infof("[%s:%s] is manifest.v2", m.source, m.tag)
			if err := m.initImageListByV2(); err != nil {
				return fmt.Errorf("Mirror: %w", err)
			}
		default:
			return u.ErrInvalidMediaType
		}
	case 1:
		logrus.WithField("MID", m.mID).
			Infof("[%s:%s] is manifest.v1", m.source, m.tag)
		if err := m.initImageListByV1(); err != nil {
			return fmt.Errorf("Mirror: %w", err)
		}
	default:
		return u.ErrInvalidSchemaVersion
	}

	for _, img := range m.images {
		if err := img.Copy(); err != nil {
			logrus.WithFields(logrus.Fields{"MID": m.mID}).Error(err.Error())
		}
	}

	// If the source manifest list does not equal to the dest manifest list
	if !m.compareSourceDestManifest() {
		logrus.WithField("MID", m.mID).
			Info("Creating dest manifest list...")
		if err := m.updateDestManifest(); err != nil {
			return err
		}
	} else {
		logrus.WithField("MID", m.mID).
			Info("Dest manifest list already exists, no need to recreate")
	}

	logrus.WithField("MID", m.mID).
		Infof("Successfully copied %s:%s => %s:%s.",
			m.source, m.tag, m.destination, m.tag)

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

func (m *Mirror) HasArch(a string) bool {
	return slices.Contains(m.availableArchList, a)
}

func (m *Mirror) ImageNum() int {
	return len(m.images)
}

func (m *Mirror) AppendImage(img image.Imagerer) {
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
	schema, ok := u.ReadJsonInt(m.destManifest, "schemaVersion")
	if !ok {
		return digests
	}

	switch schema {
	case 1:
		return digests
	case 2:
		mediaType, ok := u.ReadJsonString(m.destManifest, "mediaType")
		if !ok {
			return digests
		}
		switch mediaType {
		case u.MediaTypeManifestListV2:
			manifests, ok := u.ReadJsonSubArray(m.destManifest, "manifests")
			if !ok {
				return digests
			}
			for _, v := range manifests {
				manifest, ok := v.(map[string]interface{})
				if !ok {
					return digests
				}
				digest, ok := u.ReadJsonString(manifest, "digest")
				if !ok {
					return digests
				}
				digests = append(digests, digest)
			}
			return digests
		case u.MediaTypeManifestV2:
			digest, ok := u.ReadJsonString(m.destManifest, "digest")
			if !ok {
				return digests
			}
			digests = append(digests, digest)
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

// Failed method gets the number of images copy failed
func (m *Mirror) Failed() int {
	var num int = 0
	for _, img := range m.images {
		if !img.Copied() {
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

	// Get destination manifest list
	inspectDestImage := fmt.Sprintf("docker://%s:%s", m.destination, m.tag)
	out, err = registry.SkopeoInspect(inspectDestImage, "--raw")
	if err != nil {
		// destination image not found, this error is expected
		return nil
	}

	if err = json.NewDecoder(strings.NewReader(out)).
		Decode(&m.destManifest); err != nil {
		return fmt.Errorf("decode destination manifest json: %w", err)
	}

	return nil
}

func (m *Mirror) sourceManifestSchemaVersion() (int, error) {
	schemaVersion, ok := u.ReadJsonInt(m.sourceManifest, "schemaVersion")
	if !ok {
		return 0, fmt.Errorf(
			"SourceManifestSchemaVersion read schemaVersion: %w",
			u.ErrReadJsonFailed)
	}
	return schemaVersion, nil
}

func (m *Mirror) sourceManifestMediaType() (string, error) {
	mediaType, ok := u.ReadJsonString(m.sourceManifest, "mediaType")
	if !ok {
		return "", fmt.Errorf("SourceManifestMediaType read mediaType: %w",
			u.ErrReadJsonFailed)
	}
	return mediaType, nil
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

	logrus.WithField("MID", m.mID).Debug("start initImageListByListV2")
	manifests, ok := u.ReadJsonSubArray(m.sourceManifest, "manifests")
	if !ok {
		// unable to read manifests list, return error of this image
		return fmt.Errorf("reading manifests: %w", u.ErrReadJsonFailed)
	}

	// var images int = 0
	for _, v := range manifests {
		if manifest, ok = v.(map[string]interface{}); !ok {
			continue
		}
		if digest, ok = u.ReadJsonString(manifest, "digest"); !ok {
			continue
		}
		logrus.WithField("MID", m.mID).Debugf("digest: %s", digest)
		if platform, ok = u.ReadJsonSubObj(manifest, "platform"); !ok {
			continue
		}
		if arch, ok = u.ReadJsonString(platform, "architecture"); !ok {
			continue
		}
		// variant is empty string if not found
		variant, _ = u.ReadJsonString(platform, "variant")
		if !slices.Contains(m.availableArchList, arch) {
			logrus.WithField("MID", m.mID).
				Debugf("skip copy image %s arch %s", m.source, arch)
			continue
		}
		logrus.WithField("MID", m.mID).Debugf("arch: %s", arch)
		osType, _ = u.ReadJsonString(platform, "os")

		// create a new image object and append it into image list
		image := image.NewImage(&image.ImageOptions{
			Source:              m.source,
			Destination:         m.destination,
			Tag:                 m.tag,
			Arch:                arch,
			Variant:             variant,
			OS:                  osType,
			Digest:              digest,
			SourceSchemaVersion: 2,
			SourceMediaType:     u.MediaTypeManifestListV2,
			MID:                 m.mID,
		})
		m.AppendImage(image)
		// images++
	}

	// if images == 0 {
	// 	logrus.WithField("MID", m.mID).Debug("image [%s] does not have arch %v",
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
		arch   string
		osType string
		config map[string]interface{}
		digest string
		ok     bool
	)

	var sourceOciConfig map[string]interface{}
	json.NewDecoder(strings.NewReader(out)).Decode(&sourceOciConfig)
	if arch, ok = u.ReadJsonString(sourceOciConfig, "architecture"); !ok {
		return fmt.Errorf("initImageListByV2 read architecture: %w",
			u.ErrReadJsonFailed)
	}
	osType, _ = u.ReadJsonString(sourceOciConfig, "os")

	if config, ok = u.ReadJsonSubObj(m.sourceManifest, "config"); !ok {
		return fmt.Errorf("initImageListByV2 read config: %w",
			u.ErrReadJsonFailed)
	}
	if digest, ok = u.ReadJsonString(config, "digest"); !ok {
		return fmt.Errorf("initImageListByV2 read digest: %w",
			u.ErrReadJsonFailed)
	}

	if !slices.Contains(m.availableArchList, arch) {
		logrus.WithField("MID", m.mID).
			Debugf("skip copy image %s arch %s", m.source, arch)
	}

	// create a new image object and append it into image list
	image := image.NewImage(&image.ImageOptions{
		Source:              m.source,
		Destination:         m.destination,
		Tag:                 m.tag,
		Arch:                arch,
		Variant:             "",
		OS:                  osType,
		Digest:              digest,
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

	if arch, ok = u.ReadJsonString(m.sourceManifest, "architecture"); !ok {
		return fmt.Errorf("read architecture failed: %w", u.ErrReadJsonFailed)
	}
	if !slices.Contains(m.availableArchList, arch) {
		logrus.WithField("MID", m.mID).
			Debugf("skip copy image %s arch %s", m.source, arch)
	}

	if osType, ok = u.ReadJsonString(sourceInfo, "Os"); !ok {
		return fmt.Errorf("read Os failed: %w", u.ErrReadJsonFailed)
	}

	// create a new image object and append it into image list
	img := image.NewImage(&image.ImageOptions{
		Source:              m.source,
		Destination:         m.destination,
		Tag:                 m.tag,
		Arch:                arch,
		Variant:             "",
		OS:                  osType,
		Digest:              "", // schemaVersion 1 does not have digest
		SourceSchemaVersion: 1,
		SourceMediaType:     "", // schemaVersion 1 does not have mediaType
		MID:                 m.mID,
	})
	m.AppendImage(img)

	return nil
}

func (m *Mirror) compareSourceDestManifest() bool {
	if m.destManifest == nil {
		// dest image does not exist, return false
		logrus.WithField("MID", m.mID).
			Debug("compareSourceDestManifest: dest manifest does not exist")
		return false
	}
	schema, ok := u.ReadJsonInt(m.destManifest, "schemaVersion")
	if !ok {
		// read json failed, return false
		logrus.WithField("MID", m.mID).
			Debug("compareSourceDestManifest: read schemaVersion failed")
		return false
	}
	switch schema {
	// The destination manifest list schemaVersion should be 2
	case 1:
		logrus.WithField("MID", m.mID).
			Debug("compareSourceDestManifest: dest schemaVersion is 1")
		return false
	case 2:
		mediaType, ok := u.ReadJsonString(m.destManifest, "mediaType")
		if !ok {
			return false
		}
		switch mediaType {
		case u.MediaTypeManifestListV2:
			// Compare the source image digest list and dest image digest list
			srcDigests := m.SourceDigests()
			dstDigests := m.DestinationDigests()
			logrus.WithField("MID", m.mID).
				Debug("compareSourceDestManifest: ")
			logrus.WithField("MID", m.mID).
				Debugf("  srcDigests: %v", srcDigests)
			logrus.WithField("MID", m.mID).
				Debugf("  dstDigests: %v", dstDigests)
			return slices.Compare(srcDigests, dstDigests) == 0
		case u.MediaTypeManifestV2:
			// The destination manifest mediaType should be 'manifest.list.v2'
			logrus.WithField("MID", m.mID).
				Debug("compareSourceDestManifest: dest mediaType is m.v2")
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
		if !img.Copied() {
			continue
		}
		copiedImage := fmt.Sprintf("%s:%s", img.Destination(), img.CopiedTag())
		args = append(args, copiedImage)
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
