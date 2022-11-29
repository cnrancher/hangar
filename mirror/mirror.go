package mirror

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

func MirrorImages(fileName, arches, sourceReg, destReg, loginURL string) {
	username := os.Getenv("REGISTRY_USERNAME")
	passwd := os.Getenv("REGISTRY_PASSWORD")
	if username == "" || passwd == "" {
		logrus.Fatal("REGISTRY_USERNAME and REGISTRY_PASSWORD environment variable not set!")
		// TODO: read username and password by commandline
	}
	token, err := registry.LoginToken(destReg, username, passwd)
	if err != nil {
		logrus.Fatalf("failed to login: %v", err.Error())
	}
	if token == "" {
		return
	}

	var scanner *bufio.Scanner
	if fileName == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		logrus.Info("Reading '<SOURCE> <DESTINATION> <TAG>' from stdin")
		logrus.Info("Use Ctrl+D to exit...")
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
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
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
			continue
		}
		logrus.Debugf("SOURCE: [%v] DEST: [%v] TAG: [%v]", v[0], v[1], v[2])

		// TODO: ensure that source specifies an explicit registry and repository

		// override destination registry if set

		// override destination org/user if set

		// source, err := url.Parse(v[0])
		// if err != nil {
		// 	logrus.Errorf(err.Error())
		// }
		// dest, err := url.Parse(v[1])
		// if err != nil {
		// 	logrus.Errorf(err.Error())
		// }

		err = mirrorImage(v[0], v[1], v[2], strings.Split(arches, ","))
		if err != nil {
			logrus.Errorf("Failed copying image for %s\n", v[1])
			logrus.Error(err.Error())
			continue
		}
	}
}

func copyIfChanged(sourceRef, destRef, arch, args string) error {

	// Inspect the source image info
	inspectImg := fmt.Sprintf("docker://%s", sourceRef)
	sourceManifestBuff, err := registry.SkopeoInspectRaw(inspectImg)
	if err != nil {
		return fmt.Errorf("copyIfChanged: %w", err)
	}
	fmt.Println(sourceManifestBuff.String())
	// var sourceManifest map[string]interface{}
	// json.NewDecoder(buff).Decode(&sourceManifest)

	inspectImg = fmt.Sprintf("docker://%s", destRef)
	destManifestBuff, err := registry.SkopeoInspectRaw(inspectImg)
	if err != nil {
		return fmt.Errorf("copyIfChanged: %w", err)
	}
	fmt.Println(destManifestBuff.String())

	// TODO: calculate sha256sum for manifest and compare it.

	return nil
}

func setRepoDescription(sourceSpec, destSpec string) error {
	return nil
}

func mirrorImage(source, dest, tag string, archList []string) error {
	if archList == nil || len(archList) <= 0 {
		return fmt.Errorf("invalid arch list")
	}

	inspectImg := fmt.Sprintf("docker://%s:%s", source, tag)
	buff, err := registry.SkopeoInspectRaw(inspectImg)
	if err != nil {
		return fmt.Errorf("mirrorImage: %w", err)
	}

	var sourceInfoMap map[string]interface{}
	json.NewDecoder(buff).Decode(&sourceInfoMap)
	// Get source image schemaVersion and mediaType
	schemaVersion, ok := u.ReadJsonIntVal(sourceInfoMap, "schemaVersion")
	if !ok {
		return fmt.Errorf("reading schemaVersion: %w", u.ErrReadJsonFailed)
	}
	mediaType, ok := u.ReadJsonStringVal(sourceInfoMap, "mediaType")
	if !ok {
		return fmt.Errorf("reading mediaType: %w", u.ErrReadJsonFailed)
	}
	logrus.Debugf("schemaVersion: %v", schemaVersion)
	logrus.Debugf("mediaType: %v", mediaType)

	if schemaVersion == 2 {
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
			if err != nil {
				return fmt.Errorf("mirrorImage: %w", err)
			}

			// if there is no image copied to dest registry, return an error
			copiedImageNum := 0
			for _, v := range manifestList {
				m := v.(map[string]interface{})
				logrus.Debugf("manifest: %v", m)

				digest, ok := u.ReadJsonStringVal(m, "digest")
				if !ok {
					// digest not found in manifest list,
					// failed to copy this image, skip
					continue
				}
				logrus.Debugf("---- digest: %s", digest)
				platform, ok := u.ReadJsonSubObj(m, "platform")
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
				if !slices.Contains(archList, arch) {
					logrus.Infof("skip copy image %s arch %s", source, arch)
					continue
				}
				logrus.Debugf("---- arch: %s", arch)
				// os, ok := u.ReadJsonStringVal(platform, "os")

				// retag the destination image with ARCH information
				var destFmt string
				// if this platform has variant and the arch is not arm64v8
				// (arm64 only have one variant v8 so we skip it)
				variant, ok := u.ReadJsonStringVal(platform, "variant")
				if ok && arch != "arm64" && variant != "" {
					// ${DEST}:${TAG}-${ARCH}${VARIANT}
					destFmt = fmt.Sprintf("%s:%s-%s%s",
						dest, tag, arch, variant)
				} else {
					// ${DEST}:${TAG}-${ARCH}
					destFmt = fmt.Sprintf("%s:%s-%s",
						dest, tag, arch)
				}
				// ${SOURCE}@${DIGEST}
				srcFmt := fmt.Sprintf("%s@%s", source, digest)
				logrus.Debugf("---- srcFmt: %s", srcFmt)
				logrus.Debugf("---- destFmt: %s", destFmt)
				err = copyIfChanged(srcFmt, destFmt, arch, "")
				if err != nil {
					// TODO:
				}

				// If image copied successfully
				copiedImageNum++
			}
		case "application/vnd.docker.distribution.manifest.v2+json":
			// Standalone manifests don't include architecture info,
			// we have to get that from the image config

		default:
		}
	} else if schemaVersion == 1 {

	} else {
		return fmt.Errorf("unknown schemaVersion: %v", schemaVersion)
	}

	return nil
}
