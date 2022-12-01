package mirror

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

type Mirrorer struct {
	source     string
	destnation string
	tag        string
	archList   []string

	failedImage []string
}

func MirrorImages(fileName, arches, sourceRegOverride, destRegOverride string) {
	username := os.Getenv("DOCKER_USERNAME")
	passwd := os.Getenv("DOCKER_PASSWORD")
	regUrl := os.Getenv("DOCKER_REGISTRY")
	if username == "" || passwd == "" {
		logrus.Fatal("DOCKER_USERNAME and DOCKER_PASSWORD environment variable not set!")
		// TODO: read username and password from stdin
	}

	if destRegOverride != "" {
		regUrl = destRegOverride
	}

	// execute docker login command
	err := registry.Login(regUrl, username, passwd)
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
		var err error
		line := scanner.Text()
		// Ignore empty/comment line
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			if usingStdin {
				fmt.Printf(">>>")
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
				fmt.Printf(">>>")
			}
			continue
		}

		mirrorer := Mirrorer{
			source:     constructureRegistry(v[0], sourceRegOverride),
			destnation: constructureRegistry(v[1], destRegOverride),
			tag:        v[2],
			archList:   strings.Split(arches, ","),
		}
		logrus.Infof("SOURCE: [%v] DEST: [%v] TAG: [%v]",
			mirrorer.source, mirrorer.destnation, mirrorer.tag)

		err = mirrorer.mirrorImage()
		if err != nil {
			logrus.Errorf("Failed for copy image %s\n", mirrorer.source)
			logrus.Error(err.Error())
			// TODO: append the image to the list which is copy failed
			if usingStdin {
				fmt.Printf(">>>")
			}
			continue
		}
		if usingStdin {
			fmt.Printf(">>>")
		}
	}
	if usingStdin {
		fmt.Println()
	}
}

func setRepoDescription(sourceSpec, destSpec string) error {
	return nil
}

func (m *Mirrorer) mirrorImage() error {
	if m.archList == nil || len(m.archList) == 0 {
		return fmt.Errorf("invalid arch list")
	}

	inspectSrcImg := fmt.Sprintf("docker://%s:%s", m.source, m.tag)
	buff, err := registry.SkopeoInspectRaw(inspectSrcImg)
	if err != nil {
		return fmt.Errorf("mirrorImage: %w", err)
	}

	var sourceInfoMap map[string]interface{}
	json.NewDecoder(buff).Decode(&sourceInfoMap)
	// Get source image schemaVersion and mediaType
	srcSchemaVersion, ok := u.ReadJsonIntVal(sourceInfoMap, "schemaVersion")
	if !ok {
		return fmt.Errorf("reading schemaVersion: %w", u.ErrReadJsonFailed)
	}
	mediaType, ok := u.ReadJsonStringVal(sourceInfoMap, "mediaType")
	if !ok {
		return fmt.Errorf("reading mediaType: %w", u.ErrReadJsonFailed)
	}
	logrus.Debugf("schemaVersion: %v", srcSchemaVersion)
	logrus.Debugf("mediaType: %v", mediaType)

	if srcSchemaVersion == 2 {
		manifestList, ok := u.ReadJsonSubArray(sourceInfoMap, "manifests")
		if !ok {
			// unable to read manifests list, return error of this image
			return fmt.Errorf("reading manifests: %w", u.ErrReadJsonFailed)
		}
		switch mediaType {
		case "application/vnd.docker.distribution.manifest.list.v2+json":
			// Handle manifest lists by copying all the architectures
			// (and their variants) out to individual suffixed tags in the
			// destination, then recombining them into a single manifest list
			// on the bare tags.

			m.copyByManifestListV2(manifestList)
		case "application/vnd.docker.distribution.manifest.v2+json":
			// Standalone manifests don't include architecture info,
			// we have to get that from the image config

		default:
		}
	} else if srcSchemaVersion == 1 {

	} else {
		return fmt.Errorf("unknown schemaVersion: %v", srcSchemaVersion)
	}

	return nil
}

// constructureRegistry will re-construct the image url:
//
// If `registryOverride` is "", example:
// nginx --> docker.io/nginx (add docker.io prefix)
// reg.io/nginx --> reg.io/nginx (nothing changed)
// reg.io/user/nginx --> reg.io/user/nginx (nothing changed)
//
// If `registryOverride` set, example:
// nginx --> ${registryOverride}/nginx (add ${registryOverride} prefix)
// reg.io/nginx --> ${registryOverride}/nginx (set registry ${registryOverride})
// reg.io/user/nginx --> ${registryOverride}/user/nginx (same as above)
func constructureRegistry(image, registryOverride string) string {
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

	var sBuffer bytes.Buffer
	for _, v := range s {
		sBuffer.WriteString(v + "/")
	}
	return strings.TrimRight(sBuffer.String(), "/")
}

// copyByManifestListV2
func (m *Mirrorer) copyByManifestListV2(manifestList []interface{}) ([]interface{}, error) {
	var err error
	// if there is no image copied to dest registry, return an error
	copiedImageNum := 0
	for _, v := range manifestList {
		manifest := v.(map[string]interface{})
		// logrus.Debugf("manifest: %v", manifest)

		digest, ok := u.ReadJsonStringVal(manifest, "digest")
		if !ok {
			// digest not found in manifest list,
			// failed to copy this image, skip
			continue
		}
		logrus.Debugf("---- digest: %s", digest)
		platform, ok := u.ReadJsonSubObj(manifest, "platform")
		if !ok {
			// platform not found in this manifest,
			// failed to copy this image, skip
			continue
		}
		arch, ok := u.ReadJsonStringVal(platform, "architecture")
		if !ok {
			// platform does not have architecture attrib,
			// failed to copy this image, skip
			continue
		}
		if !slices.Contains(m.archList, arch) {
			logrus.Debugf("skip copy image %s arch %s", m.source, arch)
			continue
		}
		os, _ := u.ReadJsonStringVal(platform, "os")

		// retag the destination image with ARCH information
		var destFmt string
		// if this platform has variant and the arch is not arm64v8
		// (arm64 only have one variant v8 so we skip it)
		variant, ok := u.ReadJsonStringVal(platform, "variant")
		if ok && arch != "arm64" && variant != "" {
			// ${DEST}:${TAG}-${ARCH}${VARIANT}
			destFmt = fmt.Sprintf("%s:%s-%s%s",
				m.destnation, m.tag, arch, variant)
		} else {
			// ${DEST}:${TAG}-${ARCH}
			destFmt = fmt.Sprintf("%s:%s-%s",
				m.destnation, m.tag, arch)
		}
		// ${SOURCE}@${DIGEST}
		srcFmt := fmt.Sprintf("%s@%s", m.source, digest)
		logrus.Debugf("---- srcFmt: %s", srcFmt)
		logrus.Debugf("---- destFmt: %s", destFmt)
		logrus.Infof("Copying image [%s] to [%s] arch [%s]",
			m.source, m.destnation, arch)
		err = m.copyIfChanged(srcFmt, destFmt, arch, os)
		if err != nil {
			logrus.Errorf("Error accured: %s", err.Error())
			continue
		}

		// If image copied successfully
		copiedImageNum++
	}

	return nil, nil
}

func (m *Mirrorer) copyIfChanged(sourceRef, destRef, arch, os string) error {
	// Inspect the source image info
	sourceDockerImage := fmt.Sprintf("docker://%s", sourceRef)
	sourceManifestBuff, err := registry.SkopeoInspectRaw(sourceDockerImage)
	if err != nil {
		// if source image not found, return error.
		return fmt.Errorf("copyIfChanged: %w", err)
	}
	// fmt.Println(sourceManifestBuff.String())

	destDockerImage := fmt.Sprintf("docker://%s", destRef)
	destManifestBuff, err := registry.SkopeoInspectRaw(destDockerImage)
	if err != nil {
		// if destination image not found, set destManifestBuff to nil
		destManifestBuff = nil
	}
	// fmt.Println(destManifestBuff.String())

	srcManifestSum := u.Sha256Sum(sourceManifestBuff.String())
	dstManifestSum := u.Sha256Sum(destManifestBuff.String())
	if srcManifestSum == dstManifestSum {
		logrus.Infof("    Unchanged: %s... == %s", sourceRef[0:20], destRef)
		logrus.Infof("    Digest   : %s", srcManifestSum)
	} else {
		logrus.Infof("    Copying: %s => %s", sourceRef, destRef)
		logrus.Infof("             %s == %s", srcManifestSum, dstManifestSum)
		registry.SkopeoCopyArchOS(
			arch, os, sourceDockerImage, destDockerImage, nil)
	}

	return nil
}
