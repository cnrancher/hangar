package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func GetLoginToken(url string, username string, passwd string) (string, error) {
	// export DOCKER_TOKEN=$(curl -s -d @- -X POST -H "Content-Type: application/json" https://hub.docker.com/v2/users/login/ <<< '{"username": "'${DOCKER_USERNAME}'", "password": "'${DOCKER_PASSWORD}'"}' | jq -r '.token')
	if url == "" {
		url = utils.DockerLoginURL
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
		return "", utils.ErrLoginFailed
	}
	logrus.Debugf("Get token: %v...", token.(string)[:20])

	return token.(string), nil
}

func DockerLogin(url string, username string, passwd string) error {
	// docker login ${DOCKER_REGISTRY} --username=${USERNAME} --password-stdin
	if url == "" {
		url = utils.DockerHubRegistry
	}
	logrus.Infof("Logging in to %v", url)

	path, err := exec.LookPath("docker")
	if err != nil {
		return utils.ErrDockerNotFound
	}
	logrus.Debugf("found docker installed at: %v", path)

	// Inspect the source image info
	var stdout bytes.Buffer
	cmd := exec.Command(
		path,
		"login",
		url,
		"-u", username,
		"--password-stdin",
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	cmd.Stdin = strings.NewReader(passwd)
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("docker login: \n%s\n%w", stdout.String(), err)
	}
	logrus.Info("Login successfully.")

	return nil
}

func DockerManifestCreate(name string, values ...string) error {
	logrus.Debug("Running docker manifest create...")
	if values == nil {
		return utils.ErrInvalidParameter
	}
	// Ensure docker installed
	path, err := exec.LookPath("docker")
	if err != nil {
		return utils.ErrDockerNotFound
	}

	// Inspect the source image info
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var args []string
	args = append(args, "manifest", "create", name)
	args = append(args, values...)
	cmd := exec.Command(path, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("DockerManifestCreate: \n%s\n%w",
			stderr.String(), err)
	}

	return nil
}
