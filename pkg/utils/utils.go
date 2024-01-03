package utils

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

var (
	ErrVersionIsEmpty   = errors.New("version is empty string")
	ErrNoAvailableImage = errors.New("no available image for specified arch and os")

	cacheDir string
)

const (
	HangarGitHubURL         = "https://github.com/cnrancher/hangar"
	DockerHubRegistry       = "docker.io"
	CacheCloneRepoDirectory = "charts-repo-cache"
	MaxWorkerNum            = 20
	MinWorkerNum            = 1
)

func init() {
	if os.Getenv("HOME") == "" {
		// Use /var/tmp/hangar_cache as cache folder.
		cacheDir = path.Join("/", "var", "tmp", "hangar_cache")
	} else {
		// Use ${HOME}/.cache/hangar_cache as cache folder
		cacheDir = path.Join(os.Getenv("HOME"), ".cache", "hangar_cache")
	}
	os.MkdirAll(cacheDir, 0755)
}

// Get the hangar cache dir.
//
// The default cache dir is `${HOME}/.cache/hangar_cache`.
// Or using `/var/tmp/hangar_cache` as cache dir if the $HOME env is missing.
func CacheDir() string {
	return cacheDir
}

func Sha256Sum(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func Base64(s string) string {
	data := []byte(s)
	return base64.StdEncoding.EncodeToString(data)
}

func DecodeBase64(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func AppendFileLine(fileName string, line string) error {
	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("AppendFileLine: %w", err)
	}
	if _, err := f.Write([]byte(line + "\n")); err != nil {
		f.Close() // ignore error; Write error takes precedence
		return fmt.Errorf("AppendFileLine: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("AppendFileLine: %w", err)
	}

	return nil
}

func SaveSlice(fileName string, data []string) error {
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("SaveSlice: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(strings.Join(data, "\n"))
	if err != nil {
		return fmt.Errorf("SaveSlice: %w", err)
	}
	return nil
}

// ConstructRegistry will re-construct the image url:
//
// If `registryOverride` is empty string, example:
//
//	nginx --> docker.io/nginx (add docker.io prefix)
//	reg.io/nginx --> reg.io/nginx (nothing changed)
//	reg.io/user/nginx --> reg.io/user/nginx (nothing changed)
//
// If `registryOverride` set, example:
//
//	nginx --> ${registryOverride}/nginx (add ${registryOverride} prefix)
//	reg.io/nginx --> ${registryOverride}/nginx (set registry ${registryOverride})
//	reg.io/user/nginx --> ${registryOverride}/user/nginx (same as above)
func ConstructRegistry(image, registryOverride string) string {
	spec := strings.Split(image, "/")
	var s = make([]string, 0)
	for _, v := range spec {
		if len(v) > 0 {
			s = append(s, v)
		}
	}
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

// ReplaceProjectName will replace the image project name:
//
// If `project` is empty string, the project name will be removed:
//
//	nginx --> nginx (nothing changed)
//	reg.io/nginx --> reg.io/nginx (nothing changed)
//	user/nginx --> nginx (remove project name)
//	reg.io/user/nginx --> reg.io/nginx (remove project name)
//
// If `project` set, example:
//
//	nginx --> ${project}/nginx (add project name)
//	user/nginx --> ${project}/nginx (replace project name)
//	reg.io/nginx --> reg.io/${project}/nginx
//	reg.io/user/nginx --> reg.io/${project}/nginx
func ReplaceProjectName(image, project string) string {
	spec := strings.Split(image, "/")
	var s = make([]string, 0)
	for _, v := range spec {
		if len(v) > 0 {
			s = append(s, v)
		}
	}
	switch len(s) {
	case 1:
		if project != "" {
			s = append([]string{project}, s...)
		}
	case 2:
		if strings.ContainsAny(s[0], ".:") || s[0] == "localhost" {
			if project != "" {
				s = []string{s[0], project, s[1]}
			}
		} else {
			if project != "" {
				s = append([]string{project}, s[1])
			} else {
				s = []string{s[1]}
			}
		}
	case 3:
		if project != "" {
			s[1] = project
		} else {
			// remove project name
			s = []string{s[0], s[2]}
		}
	}
	return strings.Join(s, "/")
}

// GetProjectName gets the project name of the image, example:
//
//	nginx -> "library"
//	docker.io/nginx -> "library"
//	library/nginx -> "library"
//	docker.io/library/nginx -> "library"
func GetProjectName(image string) string {
	spec := strings.Split(image, "/")
	var s = make([]string, 0)
	for _, v := range spec {
		if len(v) > 0 {
			s = append(s, v)
		}
	}

	switch len(s) {
	case 1:
		return "library"
	case 2:
		if strings.ContainsAny(s[0], ".:") || s[0] == "localhost" {
			return "library"
		} else {
			return s[0]
		}
	case 3:
		return s[1]
	}
	return "library"
}

// GetRegistryName gets the registry name of the image, example:
//
//	nginx -> docker.io
//	reg.io/nginx -> reg.io
//	library/nginx -> docker.io
//	reg.io/library/nginx -> reg.io
func GetRegistryName(image string) string {
	spec := strings.Split(image, "/")
	var s = make([]string, 0)
	for _, v := range spec {
		if len(v) > 0 {
			s = append(s, v)
		}
	}

	switch len(s) {
	case 1:
		return DockerHubRegistry
	case 2:
		if strings.ContainsAny(s[0], ".:") || s[0] == "localhost" {
			return s[0]
		} else {
			return DockerHubRegistry
		}
	case 3:
		return s[0]
	}
	return DockerHubRegistry
}

// GetImageName gets the image name, example:
//
//	nginx:latest -> nginx
//	reg.io/nginx:latest -> nginx
//	library/nginx:latest -> nginx
//	reg.io/library/nginx -> nginx
func GetImageName(image string) string {
	spec := strings.Split(image, "/")
	var s = make([]string, 0)
	for _, v := range spec {
		if len(v) > 0 {
			s = append(s, v)
		}
	}
	switch len(s) {
	case 1:
		if strings.Contains(s[0], ":") {
			return strings.Split(s[0], ":")[0]
		}
		return s[0]
	case 2:
		if strings.Contains(s[1], ":") {
			return strings.Split(s[1], ":")[0]
		}
		return s[1]
	case 3:
		if strings.Contains(s[2], ":") {
			return strings.Split(s[2], ":")[0]
		}
		return s[2]
	}
	return ""
}

// GetImageTag gets the image tag, example:
//
//	nginx:latest -> latest
//	reg.io/nginx:1.22 -> 1.22
//	library/nginx -> latest
//	reg.io/library/nginx -> latest
func GetImageTag(image string) string {
	spec := strings.Split(image, ":")
	var s = make([]string, 0)
	for _, v := range spec {
		if len(v) > 0 {
			s = append(s, v)
		}
	}
	switch len(s) {
	case 1:
		return "latest"
	case 2:
		return s[1]
	}
	return "latest"
}

// AddSourceToImage adds image into map[image][source]bool
func AddSourceToImage(
	imagesSet map[string]map[string]bool,
	image string,
	sources ...string,
) {
	if image == "" {
		return
	}
	if imagesSet[image] == nil {
		imagesSet[image] = make(map[string]bool)
	}
	for i := range sources {
		imagesSet[image][sources[i]] = true
	}
}

func EnsureSemverValid(v string) (string, error) {
	if !semver.IsValid(v) {
		if !semver.IsValid("v" + v) {
			return "", fmt.Errorf("%q is not a valid semver", v)
		}
		v = "v" + v
	}
	return v, nil
}

// SemverCompare compares two semvers
func SemverCompare(a, b string) (int, error) {
	if a == "" || b == "" {
		return 0, ErrVersionIsEmpty
	}
	a, err := EnsureSemverValid(a)
	if err != nil {
		return 0, err
	}
	b, err = EnsureSemverValid(b)
	if err != nil {
		return 0, err
	}
	return semver.Compare(a, b), nil
}

func SemverMajorEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	a, err := EnsureSemverValid(a)
	if err != nil {
		return false
	}
	b, err = EnsureSemverValid(b)
	if err != nil {
		return false
	}
	if semver.Major(a) != semver.Major(b) {
		return false
	}
	return true
}
func SemverMajorMinorEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	a, err := EnsureSemverValid(a)
	if err != nil {
		return false
	}
	b, err = EnsureSemverValid(b)
	if err != nil {
		return false
	}
	if semver.MajorMinor(a) != semver.MajorMinor(b) {
		return false
	}
	return true
}

func ToObj(data interface{}, into interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, into)
}

func PrintObject(a any) string {
	b, _ := json.MarshalIndent(a, "", "  ")
	return string(b)
}

func Scanf(ctx context.Context, format string, a ...any) (int, error) {
	nCh := make(chan int)
	go func() {
		n, _ := fmt.Scanf(format, a...)
		nCh <- n
	}()
	select {
	case n := <-nCh:
		return n, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func CopySystemContext(src *types.SystemContext) *types.SystemContext {
	if src == nil {
		return &types.SystemContext{}
	}
	var dest types.SystemContext = *src
	if src.ShortNameMode != nil {
		var m types.ShortNameMode = *src.ShortNameMode
		dest.ShortNameMode = &m
	}
	if src.DockerArchiveAdditionalTags != nil {
		for _, tag := range src.DockerArchiveAdditionalTags {
			dest.DockerArchiveAdditionalTags = append(dest.DockerArchiveAdditionalTags, tag)
		}
	}
	if src.DockerAuthConfig != nil {
		var c = *src.DockerAuthConfig
		dest.DockerAuthConfig = &c
	}
	if src.CompressionFormat != nil {
		var f = *src.CompressionFormat
		dest.CompressionFormat = &f
	}
	if src.CompressionLevel != nil {
		var l = *src.CompressionLevel
		dest.CompressionLevel = &l
	}
	return &dest
}

func SystemContextWithTLSVerify(sysctx *types.SystemContext, tlsVerify bool) *types.SystemContext {
	n := CopySystemContext(sysctx)
	n.OCIInsecureSkipTLSVerify = !tlsVerify
	n.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!tlsVerify)
	return n
}

func SystemContextWithSharedBlobDir(sysctx *types.SystemContext, dir string) *types.SystemContext {
	n := CopySystemContext(sysctx)
	n.OCISharedBlobDirPath = dir
	return n
}

func CopyPolicy(src *signature.Policy) (*signature.Policy, error) {
	b, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("utils.CopyPolicy: %w", err)
	}
	dest := new(signature.Policy)
	err = dest.UnmarshalJSON(b)
	if err != nil {
		return nil, fmt.Errorf("utils.CopyPolicy: %w", err)
	}
	return dest, err
}

func HTTPClientDoWithRetry(
	ctx context.Context, client *http.Client, req *http.Request,
) (*http.Response, error) {
	var resp *http.Response
	var err error
	err = retry.IfNecessary(ctx, func() error {
		logrus.Debugf("client.Do: %v", req.URL.String())
		resp, err = client.Do(req)
		return err
	}, &retry.Options{
		MaxRetry: 3,
		Delay:    time.Microsecond * 100,
	})
	return resp, err
}
