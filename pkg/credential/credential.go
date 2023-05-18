package credential

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cnrancher/hangar/pkg/credential/cache"
	"github.com/cnrancher/hangar/pkg/utils"
	dockerconfig "github.com/containers/image/v5/pkg/docker/config"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

var (
	ErrCredsStore          = errors.New("docker config use credsStore to store password")
	ErrCredsStoreUnsupport = errors.New("unsupported credsStore, only 'deskstop' supported")
)

type dockerCredDesktopOutput struct {
	ServerURL string `json:"ServerURL"`
	Username  string `json:"Username"`
	Secret    string `json:"Secret"`
}

func GetCredentialByDockerConfig(r string, cf io.Reader) (
	user, passwd string, err error) {
	if r == utils.DockerHubRegistry {
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
			return "", "", ErrCredsStoreUnsupport
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
		var credOutput dockerCredDesktopOutput
		if err := json.Unmarshal(stdout.Bytes(), &credOutput); err != nil {
			// failed to read password from credsStore
			return "", "", ErrCredsStore
		}
		if credOutput.Username == "" || credOutput.Secret == "" {
			return "", "", ErrCredsStore
		}
		logrus.Debugf("Got username %q from credsStore", credOutput.Username)
		return credOutput.Username, credOutput.Secret, nil
	}

	var authEncoded string
	auths, ok := dockerConfig["auths"].(map[string]interface{})
	if !ok {
		return "", "", utils.ErrReadJsonFailed
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

	authDecoded, err := utils.DecodeBase64(authEncoded)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode base64: %w", err)
	}
	spec := strings.Split(authDecoded, ":")
	if len(spec) != 2 {
		return "", "", fmt.Errorf("invalid username password format")
	}

	return spec[0], spec[1], nil
}

// GetRegistryPassword gets Registry credential by registry URL,
// it will try to find password from cache first,
// if not found, then try to find password from docker config.
// if passwd found in docker config, it will cache the password.
// if not found, return error
func GetRegistryCredential(url string) (
	username, passwd string, err error) {
	if url == "" {
		url = utils.DockerHubRegistry
	}
	logrus.Debugf("GetRegistryCredential: find credential of registry %q", url)
	// get password from cache
	uname, passwd := cache.Get(url)
	if uname != "" && passwd != "" {
		logrus.Debugf("got credential from cache")
		return uname, passwd, nil
	}
	// get password from config file
	systemContext := &types.SystemContext{}
	ac, err := dockerconfig.GetCredentials(systemContext, url)
	if ac.Password != "" && ac.Username != "" {
		cache.Add(ac.Username, ac.Password, url)
		return ac.Username, ac.Password, nil
	}

	// re-try to get passwd from $HOME/.docker/config.json
	cfName := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
	cf, err := os.Open(cfName)
	if err != nil {
		return "", "", fmt.Errorf("GetRegistryCredential: %w", err)
	}
	defer cf.Close()
	uname, passwd, err = GetCredentialByDockerConfig(url, cf)
	if err != nil {
		return "", "", fmt.Errorf("GetRegistryCredential: %w", err)
	}
	// store data into cache
	cache.Add(uname, passwd, url)
	return uname, passwd, nil
}
