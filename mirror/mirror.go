package mirror

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
}

type Mirror struct {
	source            string
	destination       string
	tag               string
	availableArchList []string

	sourceManifest map[string]interface{}
	destManifest   map[string]interface{}

	images []image.Imagerer
}

type MirrorOptions struct {
	Source      string
	Destination string
	Tag         string
	ArchList    []string
}

var (
	dockerUsername = os.Getenv("DOCKER_USERNAME")
	dockerPassword = os.Getenv("DOCKER_PASSWORD")
	dockerRegistry = os.Getenv("DOCKER_REGISTRY")
)

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

	logrus.Debug("start Mirror")
	// Init source and destination manifest
	if err := m.initSourceDestinationManifest(); err != nil {
		return fmt.Errorf("Mirror: %w", err)
	}

	sourceSchemaVersion, err := m.sourceManifestSchemaVersion()
	if err != nil {
		return fmt.Errorf("Mirror: %w", err)
	}
	sourceMediaType, err := m.sourceManifestMediaType()
	if err != nil {
		return fmt.Errorf("Mirror: %w", err)
	}
	logrus.Debugf("sourceSchemaVersion: %v", sourceSchemaVersion)
	logrus.Debugf("sourceMediaType: %v", sourceMediaType)

	switch sourceSchemaVersion {
	case 2:
		switch sourceMediaType {
		case u.MediaTypeManifestListV2:
			logrus.Infof("[%s:%s] is manifest.list.v2", m.source, m.tag)
			if err := m.initImageListByListV2(); err != nil {
				return fmt.Errorf("Mirror: %w", err)
			}
		case u.MediaTypeManifestV2:
			logrus.Infof("[%s:%s] is manifest.v2", m.source, m.tag)
			if err := m.initImageListByV2(); err != nil {
				return fmt.Errorf("Mirror: %w", err)
			}
		default:
			return u.ErrInvalidMediaType
		}
	case 1:
		logrus.Infof("[%s:%s] is manifest.v1", m.source, m.tag)
		if err := m.initImageListByV1(); err != nil {
			return fmt.Errorf("Mirror: %w", err)
		}
	default:
		return u.ErrInvalidSchemaVersion
	}

	for _, img := range m.images {
		if err := img.Copy(); err != nil {
			logrus.Error(err.Error())
		}
	}

	// If the source manifest list does not equal to the dest manifest list
	if !m.compareSourceDestManifest() {
		if err := m.updateDestManifest(); err != nil {
			return err
		}
	}

	logrus.Infof("Successfully copied %s:%s => %s:%s.",
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

func (m *Mirror) AppendImage(img image.Imagerer) {
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
		digest, ok := u.ReadJsonString(m.destManifest, "digest")
		if !ok {
			return digests
		}
		digests = append(digests, digest)
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

func (m *Mirror) initSourceDestinationManifest() error {
	var err error
	var buff *bytes.Buffer

	// Get source manifest list
	inspectSourceImage := fmt.Sprintf("docker://%s:%s", m.source, m.tag)
	buff, err = registry.SkopeoInspect(inspectSourceImage, "--raw")
	if err != nil {
		return fmt.Errorf("inspect source image failed: %w", err)
	}

	if err = json.NewDecoder(buff).Decode(&m.sourceManifest); err != nil {
		return fmt.Errorf("decode source manifest json: %w", err)
	}

	// Get destination manifest list
	inspectDestImage := fmt.Sprintf("docker://%s:%s", m.destination, m.tag)
	buff, err = registry.SkopeoInspect(inspectDestImage, "--raw")
	if err != nil {
		// destination image not found, this error is expected
		return nil
	}

	if err = json.NewDecoder(buff).Decode(&m.destManifest); err != nil {
		return fmt.Errorf("decode destination manifest json: %w", err)
	}

	return nil
}

func (m *Mirror) sourceManifestSchemaVersion() (int, error) {
	schemaVersion, ok := u.ReadJsonInt(m.sourceManifest, "schemaVersion")
	if !ok {
		return 0, fmt.Errorf("SourceManifestSchemaVersion: %w",
			u.ErrReadJsonFailed)
	}
	return schemaVersion, nil
}

func (m *Mirror) sourceManifestMediaType() (string, error) {
	mediaType, ok := u.ReadJsonString(m.sourceManifest, "mediaType")
	if !ok {
		return "", fmt.Errorf("SourceManifestMediaType: %w",
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

	logrus.Debug("start initImageListByListV2")
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
		logrus.Debugf("digest: %s", digest)
		if platform, ok = u.ReadJsonSubObj(manifest, "platform"); !ok {
			continue
		}
		if arch, ok = u.ReadJsonString(platform, "architecture"); !ok {
			continue
		}
		// variant is empty string if not found
		variant, _ = u.ReadJsonString(platform, "variant")
		if !slices.Contains(m.availableArchList, arch) {
			logrus.Debugf("skip copy image %s arch %s", m.source, arch)
			continue
		}
		logrus.Debugf("arch: %s", arch)
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
		})
		m.AppendImage(image)
		// images++
	}

	// if images == 0 {
	// 	logrus.Debug("image [%s] does not have arch %v",
	// 		m.source, m.availableArchList)
	// }

	return nil
}

func (m *Mirror) initImageListByV2() error {
	sourceImage := fmt.Sprintf("docker://%s:%s", m.source, m.tag)
	buff, err := registry.SkopeoInspect(sourceImage, "--raw", "--config")
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

	var sourceManifest map[string]interface{}
	json.NewDecoder(buff).Decode(&sourceManifest)
	m.sourceManifest = sourceManifest
	if arch, ok = u.ReadJsonString(m.sourceManifest, "architecture"); !ok {
		return u.ErrReadJsonFailed
	}
	osType, _ = u.ReadJsonString(m.sourceManifest, "os")
	if config, ok = u.ReadJsonSubObj(sourceManifest, "config"); !ok {
		return u.ErrReadJsonFailed
	}
	if digest, ok = u.ReadJsonString(config, "digest"); !ok {
		return u.ErrReadJsonFailed
	}

	if !slices.Contains(m.availableArchList, arch) {
		logrus.Debugf("skip copy image %s arch %s", m.source, arch)
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
	})
	m.AppendImage(image)

	return nil
}

func (m *Mirror) initImageListByV1() error {
	var (
		arch   string
		osType string
		config map[string]interface{}
		digest string
		ok     bool
	)

	if arch, ok = u.ReadJsonString(m.sourceManifest, "architecture"); !ok {
		return fmt.Errorf("read architecture failed: %w", u.ErrReadJsonFailed)
	}
	if osType, ok = u.ReadJsonString(m.sourceManifest, "os"); !ok {
		return fmt.Errorf("read os failed: %w", u.ErrReadJsonFailed)
	}
	if !slices.Contains(m.availableArchList, arch) {
		logrus.Debugf("skip copy image %s arch %s", m.source, arch)
	}
	if config, ok = u.ReadJsonSubObj(m.sourceManifest, "config"); !ok {
		return u.ErrReadJsonFailed
	}
	if digest, ok = u.ReadJsonString(config, "digest"); !ok {
		return u.ErrReadJsonFailed
	}
	// create a new image object and append it into image list
	img := image.NewImage(&image.ImageOptions{
		Source:              m.source,
		Destination:         m.destination,
		Tag:                 m.tag,
		Arch:                arch,
		Variant:             "",
		OS:                  osType,
		Digest:              digest,
		SourceSchemaVersion: 1,
		SourceMediaType:     "", // schemaVersion 1 does not have mediaType
	})
	m.AppendImage(img)

	return nil
}

func (m *Mirror) compareSourceDestManifest() bool {
	if m.destManifest == nil {
		// dest image does not exist, return false
		return false
	}
	schema, ok := u.ReadJsonInt(m.destManifest, "schemaVersion")
	if !ok {
		// read json failed, return false
		return false
	}
	switch schema {
	// The destination manifest list schemaVersion should be 2
	case 1:
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
			return slices.Compare(srcDigests, dstDigests) == 0
		case u.MediaTypeManifestV2:
			// The destination manifest mediaType should be 'manifest.list.v2'
			return false
		}
	}

	return false
}

// updateDestManifest
func (m *Mirror) updateDestManifest() error {
	sourceDigests := m.SourceDigests()
	args := []string{
		"imagetools",
		"create",
		fmt.Sprintf("--tag=%s:%s", m.destination, m.tag),
	}
	args = append(args, sourceDigests...)

	// docker buildx imagetools create --tag=registry/repository:tag <digests>
	err := registry.DockerBuildx(args...)
	if err != nil {
		return fmt.Errorf("updateDestManifest: %w", err)
	}
	return nil
}

func MirrorImages(fileName, arches, sourceRegOverride, destRegOverride string) {
	if dockerUsername == "" || dockerPassword == "" {
		logrus.Fatal("DOCKER_USERNAME and DOCKER_PASSWORD environment variable not set!")
		// TODO: read username and password from stdin
	}

	if sourceRegOverride != "" {
		logrus.Infof("Set source registry to [%s]", sourceRegOverride)
	} else {
		logrus.Infof("Set source registry to [%s]", u.DockerHubRegistry)
	}

	// Command line parameter is prior than environment variable
	if destRegOverride == "" && dockerRegistry != "" {
		destRegOverride = dockerRegistry
	}

	if destRegOverride != "" {
		logrus.Infof("Set destination registry to [%s]", destRegOverride)
	} else {
		logrus.Infof("Set destination registry to [%s]", u.DockerHubRegistry)
	}

	// execute docker login command
	err := registry.DockerLogin(destRegOverride, dockerUsername, dockerPassword)
	if err != nil {
		logrus.Fatalf("MirrorImages login failed: %v", err.Error())
	}

	var scanner *bufio.Scanner
	var usingStdin bool
	if fileName == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE> <DESTINATION> <TAG>' from stdin")
		logrus.Info("Use 'Ctrl+C' or 'Ctrl+D' to exit.")
		fmt.Printf(">>> ")
	} else {
		readFile, err := os.Open(fileName)
		if err != nil {
			fmt.Println(err)
		}
		defer readFile.Close()

		scanner = bufio.NewScanner(readFile)
		scanner.Split(bufio.ScanLines)
	}

	for scanner.Scan() {
		line := scanner.Text()
		// Ignore empty/comment line
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		var v []string
		for _, s := range strings.Split(line, " ") {
			if s != "" {
				v = append(v, s)
			}
		}
		if len(v) != 3 {
			logrus.Errorf("Invalid line format")
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		var mirrorer Mirrorer = NewMirror(&MirrorOptions{
			Source:      constructRegistry(v[0], sourceRegOverride),
			Destination: constructRegistry(v[1], destRegOverride),
			Tag:         v[2],
			ArchList:    strings.Split(arches, ","),
		})
		logrus.Infof("SOURCE: [%v] DEST: [%v] TAG: [%v]",
			mirrorer.Source(), mirrorer.Destination(), mirrorer.Tag())

		if err := mirrorer.Mirror(); err != nil {
			logrus.Errorf("Failed to copy image [%s]", mirrorer.Source())
			logrus.Error(err.Error())
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}
		if usingStdin {
			fmt.Printf(">>> ")
		}
	}
	if usingStdin {
		fmt.Println()
	}
}

// constructRegistry will re-construct the image url:
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
func constructRegistry(image, registryOverride string) string {
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
