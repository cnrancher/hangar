package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func GetLoginToken(url string, username string, passwd string) (string, error) {
	// export DOCKER_TOKEN=$(curl -s -d @- -X POST -H "Content-Type: application/json" https://hub.docker.com/v2/users/login/ <<< '{"username": "'${DOCKER_USERNAME}'", "password": "'${DOCKER_PASSWORD}'"}' | jq -r '.token')
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
		// read username and password from stdin
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
	logrus.Info("Login successfully.")

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
	checkBuildxInstalledParam := []string{"buildx"}

	_, err = execCommandFunc(path, checkBuildxInstalledParam...)
	if err != nil {
		if strings.Contains(err.Error(), "is not a docker command") {
			return fmt.Errorf("DockerBuildx: %w", u.ErrDockerBuildxNotFound)
		}
	}

	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else if u.WorkerNum == 1 {
		execCommandFunc = u.RunCommandStdoutFunc
	} else {
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// Clear the stdout
	buildxArgs := []string{"buildx"}
	buildxArgs = append(buildxArgs, args...)
	out, err := execCommandFunc(path, buildxArgs...)
	if err != nil {
		if strings.Contains(err.Error(), "certificate signed by unknown") {
			logrus.Warnf("Dest registry is using custom certificate!")
			logrus.Warnf("Add self-signed certificate to '%s'", u.ETC_SSL_FILE)
		}
		return fmt.Errorf("docker buildx: %w", err)
	}
	fmt.Print(out)

	return nil
}
