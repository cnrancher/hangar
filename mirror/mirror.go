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

type Imager interface {
}

type Image struct {
	source     string
	destnation string
	tag        string

	// available arch list
	archList []string

	copiedDigestList []string
	copiedSources    []string
	// copyFailedDigestList []string
}

var (
	dockerUsername = os.Getenv("DOCKER_USERNAME")
	dockerPassword = os.Getenv("DOCKER_PASSWORD")
	dockerRegistry = os.Getenv("DOCKER_REGISTRY")
)

func MirrorImages(fileName, arches, sourceRegOverride, destRegOverride string) {
	if dockerUsername == "" || dockerPassword == "" {
		logrus.Fatal("DOCKER_USERNAME and DOCKER_PASSWORD environment variable not set!")
		// TODO: read username and password from stdin
	}

	if sourceRegOverride != "" {
		logrus.Debugf("Set source registry to [%s]", sourceRegOverride)
	} else {
		logrus.Debugf("Set source registry to [%s]", u.DockerHubRegistry)
	}

	// Command line parameter is prior than environment variable
	if destRegOverride == "" && dockerRegistry != "" {
		destRegOverride = dockerRegistry
	}

	if destRegOverride != "" {
		logrus.Debugf("Set destination registry to [%s]", destRegOverride)
	} else {
		logrus.Debugf("Set destination registry to [%s]", u.DockerHubRegistry)
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
		var err error
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

		image := Image{
			source:     constructRegistry(v[0], sourceRegOverride),
			destnation: constructRegistry(v[1], destRegOverride),
			tag:        v[2],
			archList:   strings.Split(arches, ","),
		}
		logrus.Infof("SOURCE: [%v] DEST: [%v] TAG: [%v]",
			image.source, image.destnation, image.tag)

		err = image.mirrorImage()
		if err != nil {
			logrus.Errorf("Failed to copy image %s\n", image.source)
			logrus.Error(err.Error())
			// TODO: append the image to the list which is copy failed
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

func (img *Image) mirrorImage() error {
	if img.archList == nil || len(img.archList) == 0 {
		return fmt.Errorf("invalid arch list")
	}

	inspectSrcImg := fmt.Sprintf("docker://%s:%s", img.source, img.tag)
	buff, err := registry.SkopeoInspect(inspectSrcImg, "--raw")
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
		switch mediaType {
		case "application/vnd.docker.distribution.manifest.list.v2+json":
			// Handle manifest lists by copying all the architectures
			// (and their variants) out to individual suffixed tags in the
			// destination, then recombining them into a single manifest list
			// on the bare tags.
			logrus.Infof("[%s:%s] is manifest.list.v2", img.source, img.tag)

			manifestList, ok := u.ReadJsonSubArray(sourceInfoMap, "manifests")
			if !ok {
				// unable to read manifests list, return error of this image
				return fmt.Errorf("reading manifests: %w", u.ErrReadJsonFailed)
			}
			err = img.copyByManifestListV2(manifestList)
			if err != nil {
				return err
			}
			err = img.updateManifestList()
			if err != nil {
				return err
			}
		case "application/vnd.docker.distribution.manifest.v2+json":
			// Standalone manifests don't include architecture info,
			// we have to get that from the image config
			logrus.Infof("[%s:%s] is manifest.v2", img.source, img.tag)
			err = img.copyByManifestV2()
			if err != nil {
				return err
			}
		default:
		}
	} else if srcSchemaVersion == 1 {

	} else {
		return fmt.Errorf("unknown schemaVersion: %v", srcSchemaVersion)
	}

	return nil
}

// constructRegistry will re-construct the image url:
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

	var sBuffer bytes.Buffer
	for _, v := range s {
		sBuffer.WriteString(v + "/")
	}
	return strings.TrimRight(sBuffer.String(), "/")
}

// copyByManifestListV2
func (img *Image) copyByManifestListV2(manifestList []interface{}) error {
	var err error
	// if there is no image copied to dest registry, return an error
	copiedImageNum := 0
	for _, v := range manifestList {
		manifest := v.(map[string]interface{})
		// logrus.Debugf("manifest: %v", manifest)

		digest, ok := u.ReadJsonStringVal(manifest, "digest")
		if !ok {
			continue
		}
		logrus.Debugf("---- digest: %s", digest)
		platform, ok := u.ReadJsonSubObj(manifest, "platform")
		if !ok {
			continue
		}
		arch, ok := u.ReadJsonStringVal(platform, "architecture")
		if !ok {
			continue
		}
		if !slices.Contains(img.archList, arch) {
			logrus.Debugf("skip copy image %s arch %s", img.source, arch)
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
				img.destnation, img.tag, arch, variant)
		} else {
			// ${DEST}:${TAG}-${ARCH}
			destFmt = fmt.Sprintf("%s:%s-%s",
				img.destnation, img.tag, arch)
		}
		// ${SOURCE}@${DIGEST}
		srcFmt := fmt.Sprintf("%s@%s", img.source, digest)
		logrus.Debugf("---- srcFmt: %s", srcFmt)
		logrus.Debugf("---- destFmt: %s", destFmt)
		logrus.Infof("Copy [%s] to [%s] arch [%s]",
			img.source, img.destnation, arch)
		err = img.copyIfChanged(srcFmt, destFmt, arch, os)
		if err != nil {
			logrus.Error(err.Error())
			continue
		}

		// If image copied successfully
		copiedImageNum++
		img.copiedDigestList = append(img.copiedDigestList, digest)
	}

	if copiedImageNum == 0 {
		return fmt.Errorf("failed to copy %s", img.source)
	}
	return nil
}

func (img *Image) copyByManifestV2() error {
	inspectSrcImg := fmt.Sprintf("docker://%s:%s", img.source, img.tag)
	buff, err := registry.SkopeoInspect(
		inspectSrcImg, "--raw", "--config")
	if err != nil {
		return fmt.Errorf("copyByManifestV2: %w", err)
	}

	var sourceInfoMap map[string]interface{}
	json.NewDecoder(buff).Decode(&sourceInfoMap)

	arch, ok := u.ReadJsonStringVal(sourceInfoMap, "architecture")
	if !ok {
		return u.ErrReadJsonFailed
	}
	os, _ := u.ReadJsonStringVal(sourceInfoMap, "os")
	// config, ok := u.ReadJsonSubObj(sourceInfoMap, "config")
	// if !ok {
	// 	return u.ErrReadJsonFailed
	// }
	// digest, ok := u.ReadJsonStringVal(config, "digest")
	// if !ok {
	// 	return u.ErrReadJsonFailed
	// }

	if slices.Contains(img.archList, arch) {
		srcFmt := fmt.Sprintf("%s:%s", img.source, img.tag)
		dstFmt := fmt.Sprintf("%s:%s-%s", img.destnation, img.tag, arch)
		logrus.Infof("Copy [%s] to [%s] arch [%s]",
			img.source, img.destnation, arch)
		err := img.copyIfChanged(srcFmt, dstFmt, arch, os)
		if err != nil {
			logrus.Error(err.Error())
		}
	} else {
		logrus.Debugf("skip copy image %s arch %s", img.source, arch)
	}

	return nil
}

func (img *Image) copyIfChanged(sourceRef, destRef, arch, os string) error {
	// Inspect the source image info
	sourceDockerImage := fmt.Sprintf("docker://%s", sourceRef)
	sourceManifestBuff, err := registry.SkopeoInspect(sourceDockerImage, "--raw")
	if err != nil {
		// if source image not found, return error.
		return fmt.Errorf("copyIfChanged: %w", err)
	}
	// logrus.Debug("sourceManifest: ", sourceManifestBuff.String())

	destDockerImage := fmt.Sprintf("docker://%s", destRef)
	destManifestBuff, err := registry.SkopeoInspect(destDockerImage, "--raw")
	if err != nil {
		// if destination image not found, set destManifestBuff to nil
		destManifestBuff = nil
	}
	// logrus.Debug("destManifest: ", destManifestBuff.String())

	srcManifestSum := u.Sha256Sum(sourceManifestBuff.String())
	dstManifestSum := u.Sha256Sum(destManifestBuff.String())
	if srcManifestSum == dstManifestSum {
		logrus.Infof("    Unchanged: %s... == %s", sourceRef, destRef)
		logrus.Debugf("    source Digest: %s", srcManifestSum)
		logrus.Debugf("    dest   Digest: %s", dstManifestSum)
	} else {
		logrus.Infof("    Copying: %s => %s", sourceRef, destRef)
		logrus.Infof("             %s => %s", srcManifestSum, dstManifestSum)
		return registry.SkopeoCopyArchOS(
			arch, os, sourceDockerImage, destDockerImage)
	}

	return nil
}

// updateManifestList
func (img *Image) updateManifestList() error {
	destImage := fmt.Sprintf("docker://%s:%s", img.destnation, img.tag)
	buff, err := registry.SkopeoInspect(destImage, "--raw")
	var destInfoMap map[string]interface{}
	if err != nil {
		// this error is expected if the dest image manifest not exist
		buff = nil
	}
	// get all digists of destination image from manifest list
	json.NewDecoder(buff).Decode(&destInfoMap)
	manifests, ok := u.ReadJsonSubArray(destInfoMap, "manifests")
	if !ok {
		return fmt.Errorf("read manifests failed: %w", u.ErrReadJsonFailed)
	}
	var destDigistList []string
	for _, m := range manifests {
		digest, ok := u.ReadJsonStringVal(m.(map[string]interface{}), "digest")
		if !ok {
			return fmt.Errorf("read digest failed: %w", u.ErrReadJsonFailed)
		}
		destDigistList = append(destDigistList, digest)
	}
	if slices.Compare(destDigistList, img.copiedDigestList) != 0 {
		// destnation manifest digest list is not same with copied images
		// TODO:
	}

	return nil
}
