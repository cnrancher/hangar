package utils

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrReadJsonFailed       = errors.New("failed to read value from json")
	ErrSkopeoNotFound       = errors.New("skopeo not found")
	ErrDockerNotFound       = errors.New("docker not found")
	ErrLoginFailed          = errors.New("login failed")
	ErrNoAvailableImage     = errors.New("no image available for specified arch list")
	ErrInvalidParameter     = errors.New("invalid parameter")
	ErrInvalidMediaType     = errors.New("invalid media type")
	ErrInvalidSchemaVersion = errors.New("invalid schema version")
	ErrNilPointer           = errors.New("nil pointer")
	ErrDockerBuildxNotFound = errors.New("docker buildx not found")
	ErrDirNotEmpty          = errors.New("directory is not empty")
)

const (
	DockerLoginURL          = "https://hub.docker.com/v2/users/login/"
	DockerHubRegistry       = "docker.io"
	MediaTypeManifestListV2 = "application/vnd.docker.distribution.manifest.list.v2+json"
	MediaTypeManifestV2     = "application/vnd.docker.distribution.manifest.v2+json"
	SavedImageListFile      = "saved-images-list.json"
)

var (
	// worker number of mirrorer
	MirrorerJobNum = 1
)

func Sha256Sum(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func IsDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// read in ONLY one file
	_, err = f.Readdir(1)

	// and if the file is EOF... well, the dir is empty.
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func AppendFileLine(fileName string, line string) error {
	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("AppendFileLine: %w", err)
	}
	if _, err := f.Write([]byte(line)); err != nil {
		f.Close() // ignore error; Write error takes precedence
		return fmt.Errorf("AppendFileLine: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("AppendFileLine: %w", err)
	}

	return nil
}

func GetAbsPath(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("GetAbsPath: dir is empty")
	}
	if !filepath.IsAbs(dir) {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("StartSave: os.Getwd failed: %w", err)
		}
		dir = filepath.Join(currentDir, dir)
		return dir, nil
	}
	return dir, nil
}

func EnsureDirExists(directory string) error {
	info, err := os.Stat(directory)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(directory, 0755); err != nil {
				return fmt.Errorf("StartSave: %w", err)
			}
		} else {
			return fmt.Errorf("StartSave: %w", err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("StartSave: '%s' is not a directory", directory)
	}
	return nil
}

func SaveJson(data interface{}, fileName string) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("SaveJson: %w", err)
	}
	fileName, err = GetAbsPath(fileName)
	if err != nil {
		return fmt.Errorf("SaveJson: %w", err)
	}
	savedImageFile, err := os.OpenFile(fileName,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("SaveJson: %w", err)
	}
	savedImageFile.Write(jsonBytes)
	err = savedImageFile.Close()
	if err != nil {
		return fmt.Errorf("SaveJson: %w", err)
	}
	return nil
}
