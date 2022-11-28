package mirror

import (
	"os"

	"cnrancher.io/image-tools/docker"
	"github.com/sirupsen/logrus"
)

func MirrorImage(fileName, archList string) {
	username := os.Getenv("DOCKER_USERNAME")
	passwd := os.Getenv("DOCKER_PASSWORD")
	token, err := docker.Login("", username, passwd)
	if err != nil {
		logrus.Fatalf("failed to login: %v", err.Error())
	}
	if token == "" {
		// TODO:
	}
}
