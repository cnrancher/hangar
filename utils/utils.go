package utils

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"golang.org/x/term"
)

var (
	EnvDockerUsername = os.Getenv("DOCKER_USERNAME")
	EnvDockerPassword = os.Getenv("DOCKER_PASSWORD")
	EnvDockerRegistry = os.Getenv("DOCKER_REGISTRY")
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
	CacheImageDirectory     = "saved-image-cache"
	ETC_SSL_FILE            = "/etc/ssl/certs/ca-certificates.crt"
	MAX_WORKER_NUM          = 20
	MIN_WORKER_NUM          = 1
)

var (
	// worker number of mirrorer
	WorkerNum = 1
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

func DeleteIfExist(name string) error {
	_, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			// if does not exist, return
			return nil
		} else {
			return fmt.Errorf("DeleteIfExist: %w", err)
		}
	}

	if err := os.RemoveAll(name); err != nil {
		return fmt.Errorf("DeleteIfExist: %s", err)
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

// If using stdin, the worker num should be 1,
// if not using stdin, worker num should >= 1 && <= 20.
func CheckWorkerNum(usingStdin bool, num *int) {
	if usingStdin {
		if *num != 1 {
			logrus.Warn("Async mode not supported in stdin mode")
			logrus.Warn("Set jobs num back to 1")
			*num = 1
		}
	} else {
		if *num > MAX_WORKER_NUM {
			logrus.Warn("Worker count should be <= 20")
			logrus.Warn("Change worker count to 20")
			*num = MAX_WORKER_NUM
		} else if *num < MIN_WORKER_NUM {
			logrus.Warn("Invalid worker count")
			logrus.Warn("Change worker count to 1")
			*num = MIN_WORKER_NUM
		}
	}
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
			s = append([]string{DockerHubRegistry}, s...)
		}
	}

	return strings.Join(s, "/")
}

func ReadUsernamePasswd() (username, passwd string, err error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, err = reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	fmt.Print("Password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", "", err
	}
	fmt.Println()

	password := string(bytePassword)
	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

func CheckCacheDirEmpty() error {
	// Check cache image directory
	ok, err := IsDirEmpty(CacheImageDirectory)
	if err != nil {
		logrus.Panic(err)
	}
	if !ok {
		logrus.Warnf("Cache folder: '%s' is not empty!", CacheImageDirectory)
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Delete it before start save image? [Yes/No] ")
		for {
			text, _ := reader.ReadString('\n')
			if len(text) == 0 {
				continue
			}
			if text[0] == 'Y' || text[0] == 'y' {
				break
			} else {
				return fmt.Errorf("'%s': %w",
					CacheImageDirectory, ErrDirNotEmpty)
			}
		}
		if err := DeleteIfExist(CacheImageDirectory); err != nil {
			return err
		}
	}
	if err = EnsureDirExists(CacheImageDirectory); err != nil {
		return err
	}
	return nil
}
