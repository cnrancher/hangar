package registry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

type DockerCredDesktopOutput struct {
	ServerURL string `json:"ServerURL"`
	Username  string `json:"Username"`
	Secret    string `json:"Secret"`
}

type DockerPasswordCache struct {
	Username string
	Password string
	Registry string
}

var dockerPasswordCache = make([]DockerPasswordCache, 0)

func GetLoginToken(url string, username string, passwd string) (string, error) {
	if url == "" {
		url = u.DockerLoginURL
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		// add https prefix
		url = "https://" + url
	}
	logrus.Infof("Get token from %v", url)

	values := map[string]string{"username": username, "password": passwd}
	json_data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("LoginToken: %w", err)
	}

	// send a json post request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(json_data))
	if err != nil {
		return "", fmt.Errorf("LoginToken: %w", err)
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	token, ok := res["token"]
	if !ok {
		return "", u.ErrLoginFailed
	}
	logrus.Debugf("Get token: %v...", token.(string)[:20])

	return token.(string), nil
}

// DockerLogin executes
// 'docker login <registry> --username=<user> --password-stdin'
func DockerLogin(url, username, password string) error {
	if url == "" {
		url = u.DockerHubRegistry
	}
	var stdout bytes.Buffer
	cmd := exec.Command(
		DockerPath,
		"login",
		url,
		"-u", username,
		"--password-stdin",
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	cmd.Stdin = strings.NewReader(password)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker login: \n%s\n%w", stdout.String(), err)
	}

	// Login succeed, store registry, username, passwd into cache
	var cached bool = false
	for _, v := range dockerPasswordCache {
		if v.Password == password && v.Username == username &&
			v.Registry == url {
			// data already cached, skip
			cached = true
		}
	}
	if !cached {
		dockerPasswordCache = append(dockerPasswordCache, DockerPasswordCache{
			Username: username,
			Password: password,
			Registry: url,
		})
	}

	return nil
}

// DockerBuildx executes 'docker buildx args...' command
func DockerBuildx(args ...string) error {
	var execCommandFunc u.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else {
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// ensure docker-buildx is installed
	err := execCommandFunc(DockerPath, nil, nil, "buildx")
	if err != nil {
		if strings.Contains(err.Error(), "is not a docker command") {
			return fmt.Errorf("DockerBuildx: %w", u.ErrDockerBuildxNotFound)
		}
	}

	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else {
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// Clear the stdout
	buildxArgs := []string{"buildx"}
	buildxArgs = append(buildxArgs, args...)
	var out io.Writer = nil
	if u.WorkerNum == 1 {
		out = os.Stdout
	}
	err = execCommandFunc(DockerPath, nil, out, buildxArgs...)
	if err != nil {
		if strings.Contains(err.Error(), "certificate signed by unknown") {
			logrus.Warnf("Dest registry is using custom certificate!")
			logrus.Warnf("Add self-signed certificate to '%s'", u.ETC_SSL_FILE)
		}
		return fmt.Errorf("docker buildx: %w", err)
	}

	return nil
}

func GetDockerPasswdByConfig(r string, cf io.Reader) (
	user, passwd string, err error) {
	if r == u.DockerHubRegistry {
		r = "https://index.docker.io/v1/"
	}

	// Check already installed or not
	var dockerConfig map[string]interface{}
	if err := json.NewDecoder(cf).Decode(&dockerConfig); err != nil {
		return "", "", fmt.Errorf("decode failed: %w", err)
	}
	// if credsStore exists, do not read password from config (macOS)
	if credsStore, ok := dockerConfig["credsStore"]; ok {
		if credsStore != "desktop" {
			// unknow credsStore type
			return "", "", u.ErrCredsStoreUnsupport
		}
		logrus.Debugf("Docker config stores password from credsStore")
		// use docker-credential-desktop to get password
		var stdout bytes.Buffer
		cmd := exec.Command("docker-credential-desktop", "get")
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout
		cmd.Stdin = strings.NewReader(r)
		if err := cmd.Run(); err != nil {
			// failed to get password from credsStore
			return "", "", fmt.Errorf("registry %q not found in credsStore", r)
		}
		var credOutput DockerCredDesktopOutput
		if err := json.Unmarshal(stdout.Bytes(), &credOutput); err != nil {
			// failed to read password from credsStore
			return "", "", u.ErrCredsStore
		}
		if credOutput.Username == "" || credOutput.Secret == "" {
			return "", "", u.ErrCredsStore
		}
		logrus.Debugf("Got username %q from credsStore", credOutput.Username)
		return credOutput.Username, credOutput.Secret, nil
	}

	var authEncoded string
	auths, ok := dockerConfig["auths"].(map[string]interface{})
	if !ok {
		return "", "", u.ErrReadJsonFailed
	}
	for regName, v := range auths {
		if regName != r {
			continue
		}
		authMap, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		authEncoded = authMap["auth"].(string)
	}
	if authEncoded == "" {
		return "", "", fmt.Errorf("registry %q not found in docker config", r)
	}

	authDecoded, err := u.DecodeBase64(authEncoded)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode base64: %w", err)
	}
	spec := strings.Split(authDecoded, ":")
	if len(spec) != 2 {
		return "", "", fmt.Errorf("invalid username password format")
	}

	return spec[0], spec[1], nil
}

// GetDockerPasswordFromCache gets docker password from cache
func GetDockerPasswordFromCache(url string) (
	username, passwd string, err error) {
	if url == "" {
		url = u.DockerHubRegistry
	}
	for _, v := range dockerPasswordCache {
		if v.Registry == url {
			return v.Username, v.Password, nil
		}
	}
	return "", "", errors.New("not found")
}

// GetDockerPassword will try to find docker password from cache,
// if not found, it will try to find password from docker config.
// if passwd found in docker config, it will cache the password.
// if not found, return error
func GetDockerPassword(url string) (
	username, passwd string, err error) {
	if url == "" {
		url = u.DockerHubRegistry
	}
	// get password from cache first
	uname, passwd, err := GetDockerPasswordFromCache(url)
	if err == nil {
		return uname, passwd, nil
	}
	// if password not found in cache, get password from docker config file
	cfName := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
	cf, err := os.Open(cfName)
	if err != nil {
		return "", "", fmt.Errorf("GetDockerPassword: %w", err)
	}
	uname, passwd, err = GetDockerPasswdByConfig(url, cf)
	if err != nil {
		// failed to read passwd from config
		return "", "", fmt.Errorf("GetDockerPassword: %w", err)
	}
	// store data into cache
	dockerPasswordCache = append(dockerPasswordCache, DockerPasswordCache{
		Username: uname,
		Password: passwd,
		Registry: url,
	})
	return uname, passwd, nil
}
