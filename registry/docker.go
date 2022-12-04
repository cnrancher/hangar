package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"cnrancher.io/image-tools/utils"
	u "cnrancher.io/image-tools/utils"
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

// DockerLogin executes the `docker login <registry_url>` command
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

// DockerBuildx executes 'docker buildx ...' command
func DockerBuildx(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("DockerBuildx: %w", u.ErrInvalidParameter)
	}

	path, err := exec.LookPath("docker")
	if err != nil {
		return utils.ErrDockerNotFound
	}
	logrus.Debugf("found docker installed at: %v", path)

	var stdout *bytes.Buffer = new(bytes.Buffer)
	// Check docker buildx installed or not
	cmd := exec.Command(
		path,
		"buildx",
		"--help",
	)
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	if err := cmd.Run(); err != nil {
		if strings.Contains(stdout.String(), "is not a docker command") {
			return fmt.Errorf("DockerBuildx: %w", u.ErrDockerBuildxNotFound)
		}
		return fmt.Errorf("docker buildx: \n%s\n%w", stdout.String(), err)
	}

	// Clear the stdout
	buildxArgs := []string{"buildx"}
	buildxArgs = append(buildxArgs, args...)
	cmd = exec.Command(
		path,
		buildxArgs...,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker buildx: \n%s\n%w", stdout.String(), err)
	}

	return nil
}
