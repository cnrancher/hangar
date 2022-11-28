package mirror

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cnrancher.io/image-tools/docker"
	"cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func MirrorImages(fileName, arches, sourceReg, destReg string) {
	username := os.Getenv("DOCKER_USERNAME")
	passwd := os.Getenv("DOCKER_PASSWORD")
	if username == "" || passwd == "" {
		logrus.Fatal("DOCKER_USERNAME and DOCKER_PASSWORD environment variable not set!")
	}
	token, err := docker.Login("", username, passwd)
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
	return nil
}

func setRepoDescription(sourceSpec, destSpec string) error {
	return nil
}

func mirrorImage(source, dest, tag string, archList []string) error {
	if archList == nil || len(archList) <= 0 {
		return fmt.Errorf("invalid arch list")
	}

	// Ensure skopeo installed
	skopeoPath, err := utils.EnsureSkopeoInstalled("")
	if err != nil {
		return fmt.Errorf("mirrorImage: %w", err)
	}

	// Inspect the source image info
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(
		skopeoPath, "inspect", "--raw",
		fmt.Sprintf("docker://%s:%s", source, tag))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("mirrorImage: \n%s\n%w", stderr.String(), err)
	}

	var sourceInfoMap map[string]interface{}
	json.NewDecoder(&stdout).Decode(&sourceInfoMap)
	// Get source image schemaVersion and mediaType
	schemaVersion, ok := sourceInfoMap["schemaVersion"]
	if !ok {
		return fmt.Errorf("reading schemaVersion: %w", utils.ErrReadJsonFailed)
	}
	schemaVersion = int(schemaVersion.(float64))
	mediaType, ok := sourceInfoMap["mediaType"]
	if !ok {
		return fmt.Errorf("reading mediaType: %w", utils.ErrReadJsonFailed)
	}
	logrus.Debugf("schemaVersion: %v", schemaVersion)
	logrus.Debugf("mediaType: %v", mediaType)

	if schemaVersion == 2 {
		l, ok := sourceInfoMap["manifests"]
		if !ok {
			return fmt.Errorf("reading manifests: %w", utils.ErrReadJsonFailed)
		}
		switch mediaType {
		case "application/vnd.docker.distribution.manifest.list.v2+json":
			// Handle manifest lists by copying all the architectures
			// (and their variants) out to individual suffixed tags in the
			// destination, then recombining them into a single manifest list
			// on the bare tags.
			mList, err := getManifestList(mediaType.(string), l.([]interface{}), archList)
			if err != nil {
				return fmt.Errorf("mirrorImage: %w", err)
			}
			if len(mList) == 0 {
				logrus.Warnf("No manifest available for image %v and arch: %v",
					source, archList)
				logrus.Warnf("Skip mirror image %v", source)
				return nil
			}
			for _, m := range mList {
				logrus.Debugf("GetManifestList: %v", m)
			}
			// TODO: arch variant
			// TODO: copyIfChanged
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

// getManifestList gets the
func getManifestList(mediaType string, manifestList []interface{}, archList []string) ([]interface{}, error) {
	if manifestList == nil {
		return nil, errors.New("failed to read manifest list")
	}
	if len(manifestList) == 0 {
		return nil, errors.New("failed to read manifest list")
	}

	var mList []interface{}
	for _, arch := range archList {
		for _, m := range manifestList {
			platform, ok := m.(map[string]interface{})["platform"]
			if !ok {
				continue
			}
			a, ok := platform.(map[string]interface{})["architecture"]
			// variant, ok := platform.(map[string]interface{})["variant"]
			if a == arch {
				mList = append(mList, m)
			}
		}
	}

	return mList, nil
}
