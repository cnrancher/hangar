package utils

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"golang.org/x/mod/semver"
	"golang.org/x/term"
	"gopkg.in/yaml.v2"
)

var (
	ErrVersionIsEmpty      = errors.New("version is empty string")
	ErrNoAvailableImage    = errors.New("no available image for specified arch and os")
	ErrIsSigstoreSignature = errors.New("image is a sigstore signature")

	hangarCacheDir string // $HOME/.cache/hangar/<random>
	trivyCacheDir  string // $HOME/.cache/trivy
)

const (
	HangarGitHubURL         = "https://github.com/cnrancher/hangar"
	DockerHubRegistry       = "docker.io"
	CacheCloneRepoDirectory = "charts-repo-cache"
	MaxWorkerNum            = 20
	MinWorkerNum            = 1

	DefaultProject = "library"
	DefaultTag     = "latest"
	LocalHost      = "localhost"
)

func init() {
	var (
		base string
		err  error
	)
	base, err = os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}

	hangarCacheDir = filepath.Join(base, "hangar")
	trivyCacheDir = filepath.Join(base, "trivy")
	if err = os.MkdirAll(hangarCacheDir, 0755); err != nil {
		logrus.Warnf("mkdir %q: %v", hangarCacheDir, err)
	}
	if err = os.MkdirAll(trivyCacheDir, 0755); err != nil {
		logrus.Warnf("mkdir %q: %v", trivyCacheDir, err)
	}

	hangarCacheDir, err = os.MkdirTemp(hangarCacheDir, "*")
	if err != nil {
		logrus.Warnf("os.MkdirTemp %q: %v", hangarCacheDir, err)
	}
}

func SetupLogrus(hideTime bool) {
	formatter := &nested.Formatter{
		HideKeys:        false,
		TimestampFormat: "[15:04:05]", // hour, time, sec only
		FieldsOrder:     []string{"IMG"},
	}
	if hideTime {
		formatter.TimestampFormat = "-"
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		// Disable if the output is not terminal.
		formatter.NoColors = true
	}
	logrus.SetFormatter(formatter)
	logrus.SetOutput(io.Discard)
	logrus.AddHook(&writer.Hook{
		// Send logs with level higher than warning to stderr.
		Writer: os.Stderr,
		LogLevels: []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
		},
	})
	logrus.AddHook(&writer.Hook{
		// Send info, debug and trace logs to stdout.
		Writer: os.Stdout,
		LogLevels: []logrus.Level{
			logrus.TraceLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
		},
	})
}

func DefaultUserAgent() string {
	return "hangar/" + Version + " (github.com/cnrancher/hangar)"
}

// Get the hangar cache dir.
//
// The default cache dir is `${HOME}/.cache/hangar/<random>`.
func HangarCacheDir() string {
	return hangarCacheDir
}

// Get the teivy cache dir for trivy databases.
func TrivyCacheDir() string {
	return trivyCacheDir
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
	if strings.ContainsAny(s[0], ".:") || s[0] == LocalHost {
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
	switch l := len(s); l {
	case 1:
		if project != "" {
			s = append([]string{project}, s...)
		}
	case 2:
		if strings.ContainsAny(s[0], ".:") || s[0] == LocalHost {
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
	default:
		// The image name has slashes
		// https://github.com/cnrancher/hangar/issues/109
		if l > 3 {
			if strings.ContainsAny(s[0], ".:") || s[0] == LocalHost {
				// s[0] is the registry URL
				// s[1] is the project name to be replaced
				if project != "" {
					s[1] = project
				}
			} else {
				// s[0] is the project name
				if project != "" {
					s[0] = project
				}
			}
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

	switch l := len(s); l {
	case 1:
		return DefaultProject
	case 2:
		if strings.ContainsAny(s[0], ".:") || s[0] == LocalHost {
			return DefaultProject
		}
		return s[0]
	case 3:
		return s[1]
	default:
		// The image name has slashes
		// https://github.com/cnrancher/hangar/issues/109
		if l > 3 {
			if strings.ContainsAny(s[0], ".:") || s[0] == LocalHost {
				return s[1]
			}
			return s[0]
		}
	}
	return DefaultProject
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

	switch l := len(s); l {
	case 1:
		return DockerHubRegistry
	case 2:
		if strings.ContainsAny(s[0], ".:") || s[0] == LocalHost {
			return s[0]
		}
		return DockerHubRegistry
	case 3:
		return s[0]
	default:
		// The image name has slashes
		// https://github.com/cnrancher/hangar/issues/109
		if l > 3 {
			if strings.ContainsAny(s[0], ".:") || s[0] == LocalHost {
				return s[0]
			}
		}
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
	switch l := len(s); l {
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
	default:
		// The image name has slashes
		// https://github.com/cnrancher/hangar/issues/109
		if l > 3 {
			if strings.ContainsAny(s[0], ".:") || s[0] == LocalHost {
				return strings.Split(strings.Join(s[2:], "/"), ":")[0]
			}
			return strings.Split(strings.Join(s[1:], "/"), ":")[0]
		}
	}
	return ""
}

// GetImageTag gets the image tag, example:
//
//	nginx:latest -> latest
//	nginx:1.22 -> 1.22
//	reg.io/nginx:1.22 -> 1.22
//	library/nginx -> latest
//	reg.io/library/nginx -> latest
func GetImageTag(image string) string {
	spec := strings.Split(image, "/")
	var (
		s  = make([]string, 0)
		s1 = make([]string, 0)
	)
	for _, v := range spec {
		if len(v) > 0 {
			s = append(s, v)
		}
	}
	switch l := len(s); l {
	case 1:
		if strings.Contains(s[0], ":") {
			// Example: name:tag
			spec1 := strings.Split(s[0], ":")
			for _, v := range spec1 {
				if len(v) > 0 {
					s1 = append(s1, v)
				}
			}
		}
	case 2:
		if strings.Contains(s[1], ":") {
			// Example: library/name:tag
			spec1 := strings.Split(s[1], ":")
			for _, v := range spec1 {
				if len(v) > 0 {
					s1 = append(s1, v)
				}
			}
		}
	case 3:
		if strings.Contains(s[2], ":") {
			// Example: docker.io/library/name:tag
			spec1 := strings.Split(s[2], ":")
			for _, v := range spec1 {
				if len(v) > 0 {
					s1 = append(s1, v)
				}
			}
		}
	default:
		// The image name has slashes
		// https://github.com/cnrancher/hangar/issues/109
		if l > 3 {
			spec1 := strings.Split(s[l-1], ":")
			for _, v := range spec1 {
				if len(v) > 0 {
					s1 = append(s1, v)
				}
			}
		}
	}

	if len(s1) == 2 {
		return s1[1]
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

// Merge map[image]map[source]true from b into a
func MergeImageSourceSet(a, b map[string]map[string]bool) {
	if a == nil || b == nil {
		return
	}
	for image, sources := range b {
		if a[image] == nil {
			a[image] = make(map[string]bool)
		}
		for source := range sources {
			a[image][source] = true
		}
	}
}

// Merge map[string]true from b into a
func MergeSets(a, b map[string]bool) {
	if a == nil || b == nil {
		return
	}
	for v := range b {
		a[v] = true
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
//
// The result will be 0 if a == b, -1 if a < b, or +1 if a > b.
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

func ToJSON(a any) string {
	b, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		logrus.Warnf("failed to marsjal %T to JSON: %v", a, err)
	}
	return string(b)
}

func ToYAML(a any) string {
	b, err := yaml.Marshal(a)
	if err != nil {
		logrus.Warnf("failed to marsjal %T to JSON: %v", a, err)
	}
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

func ReadPassword(ctx context.Context) ([]byte, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, fmt.Errorf("failed to read password: stdin is is not a interactive terminal")
	}

	bCh := make(chan []byte)
	go func() {
		b, _ := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		bCh <- b
	}()
	select {
	case b := <-bCh:
		return b, nil
	case <-ctx.Done():
		return nil, ctx.Err()
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
		dest.DockerArchiveAdditionalTags = append(dest.DockerArchiveAdditionalTags, src.DockerArchiveAdditionalTags...)
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

func HTTPClientDo(
	client *http.Client, req *http.Request,
) (*http.Response, error) {
	logrus.Debugf("client.Do: %v", req.URL.String())
	resp, err := client.Do(req)
	return resp, err
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

// DetectURL detects whether the server is using HTTPS or HTTP (if in insecure mode)
// User need to call resp.Body.Close after usage.
func DetectURL(
	ctx context.Context, s string, insecure bool,
) (string, *http.Response, error) {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s, nil, nil
	}

	client := &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	u := fmt.Sprintf("https://%s", s)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", nil, fmt.Errorf("utils.DetectURL: %w", err)
	}

	pingFunc := func() (*http.Response, error) {
		resp, err := HTTPClientDo(client, req)
		if err == nil {
			return resp, nil
		}
		if !insecure {
			return resp, err
		}

		logrus.Debugf("ping %s: %v", u, err)
		// Insecure provided, try ping in HTTP mode
		u = fmt.Sprintf("http://%s", s)
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		resp, err = HTTPClientDo(client, req)
		if err != nil {
			return nil, err
		}

		logrus.Debugf("ping %s: %v", u, resp.Status)
		return resp, nil
	}

	var resp *http.Response
	err = retry.IfNecessary(ctx, func() error {
		resp, err = pingFunc()
		return err
	}, &retry.Options{
		MaxRetry: 3,
		Delay:    time.Millisecond,
	})
	if err != nil {
		return "", resp, fmt.Errorf("utils.DetectURL: %w", err)
	}
	return u, resp, nil
}

func CheckFileExistsPrompt(
	ctx context.Context, name string, autoYes bool,
) error {
	_, err := os.Stat(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	var s string
	fmt.Printf("File %q already exists! Overwrite? [y/N] ", name)
	if autoYes {
		fmt.Println("y")
	} else {
		if _, err := Scanf(ctx, "%s", &s); err != nil {
			return err
		}
		if len(s) == 0 || s[0] != 'y' && s[0] != 'Y' {
			return fmt.Errorf("file %q already exists", name)
		}
	}

	return nil
}
