package utils

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	CacheImageDirectory     = ".saved-image-cache/"
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
	info, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			// if dir does not exist, return true
			return true, nil
		} else {
			return false, fmt.Errorf("IsDirEmpty: %w", err)
		}
	} else if !info.IsDir() {
		return false, fmt.Errorf("IsDirEmpty: '%s' is not a directory", name)
	}

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
		dir = "."
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
	var jsonBytes []byte = []byte{}
	var err error

	if data != nil {
		jsonBytes, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("SaveJson: %w", err)
		}
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

// GetRepositoryName gets the repository name of the image.
// Repository name example:
// nginx -> nginx;
// library/nginx -> library/nginx;
// docker.io/nginx -> nginx;
// docker.io/library/nginx -> library/nginx;
// localhost/nginx -> nginx;
// localhost/library/nginx -> library/nginx
func GetRepositoryName(src string) (string, error) {
	v := strings.Split(src, "/")
	switch len(v) {
	case 1:
		return src, nil
	case 2:
		if strings.ContainsAny(v[0], ":.") || v[0] == "localhost" {
			return v[1], nil
		} else {
			return src, nil
		}
	case 3:
		if strings.ContainsAny(v[0], ":.") || v[0] == "localhost" {
			return strings.Join(v[1:], "/"), nil
		}
	}
	return "", errors.New("invalid format")
}
