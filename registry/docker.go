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

// DockerLogin executes the `docker login <registry_url>` command
func DockerLogin(url string) error {
	// docker login ${DOCKER_REGISTRY} --username=${USERNAME} --password-stdin
	if url == "" {
		url = u.DockerHubRegistry
	}
	logrus.Infof("Logging in to %v", url)

	path, err := exec.LookPath("docker")
	if err != nil {
		return u.ErrDockerNotFound
	}
	logrus.Debugf("found docker installed at: %v", path)

	if u.EnvDockerUsername == "" || u.EnvDockerPassword == "" {
		// read username and password from docker config file
		cPath := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
		cFile, err := os.Open(cPath)
		if err == nil {
			defer cFile.Close()
			user, pwd, err := GetDockerPasswdByConfig(url, cFile)
			if err != nil {
				if !errors.Is(err, u.ErrCredsStore) {
					logrus.Warnf(
						"Failed to get password from docker config: %s",
						err.Error())
				} else {
					logrus.Info("docker config is using credsStore, ",
						"unable to read password from docker config file")
				}
			} else if user != "" && pwd != "" {
				// read username password succeed, skip
				u.EnvDockerUsername = user
				u.EnvDockerPassword = pwd
				logrus.Infof("Get passwd of user %q from docker config",
					u.EnvDockerUsername)
			}
		} else {
			cFile.Close()
			logrus.Debug("Failed to open docker config file")
		}
	}
	// If failed to get password from docker config file
	if u.EnvDockerUsername == "" || u.EnvDockerPassword == "" {
		// read username and password from stdin
		logrus.Infof("Please input dest registry username and passwd:")
		username, passwd, err := u.ReadUsernamePasswd()
		if err != nil {
			return fmt.Errorf("DockerLogin: %w", err)
		}
		u.EnvDockerUsername = username
		u.EnvDockerPassword = passwd
	}

	var stdout bytes.Buffer
	cmd := exec.Command(
		path,
		"login",
		url,
		"-u", u.EnvDockerUsername,
		"--password-stdin",
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	cmd.Stdin = strings.NewReader(u.EnvDockerPassword)
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("docker login: \n%s\n%w", stdout.String(), err)
	}
	logrus.Info("Login succeed")

	return nil
}

// DockerBuildx executes 'docker buildx args...' command
func DockerBuildx(args ...string) error {
	path, err := exec.LookPath("docker")
	if err != nil {
		return u.ErrDockerNotFound
	}
	logrus.Debugf("found docker installed at: %v", path)

	var execCommandFunc u.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else {
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// ensure docker-buildx is installed
	err = execCommandFunc(path, nil, nil, "buildx")
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
	err = execCommandFunc(path, nil, out, buildxArgs...)
	if err != nil {
		if strings.Contains(err.Error(), "certificate signed by unknown") {
			logrus.Warnf("Dest registry is using custom certificate!")
			logrus.Warnf("Add self-signed certificate to '%s'", u.ETC_SSL_FILE)
		}
		return fmt.Errorf("docker buildx: %w", err)
	}

	return nil
}

func GetDockerPasswdByConfig(r string, cf io.Reader) (string, string, error) {
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
