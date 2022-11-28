package docker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

func Login(url string, username string, passwd string) (string, error) {
	// docker login ${DOCKER_REGISTRY:-docker.io} --username=${DOCKER_USERNAME} --password-stdin <<< ${DOCKER_PASSWORD}
	// export DOCKER_TOKEN=$(curl -s -d @- -X POST -H "Content-Type: application/json" https://hub.docker.com/v2/users/login/ <<< '{"username": "'${DOCKER_USERNAME}'", "password": "'${DOCKER_PASSWORD}'"}' | jq -r '.token')
	if url == "" {
		url = "https://hub.docker.com/v2/users/login/"
		logrus.Infof("Logging in to %v", url)
	}

	values := map[string]string{"username": username, "password": passwd}
	json_data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("Login: %w", err)
	}

	// send a json post request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(json_data))
	if err != nil {
		return "", fmt.Errorf("Login: %w", err)
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	token, ok := res["token"]
	if !ok {
		return "", errors.New("login failed")
	}
	logrus.Debugf("Get token: %v...", token.(string)[:20])

	return token.(string), nil
}
