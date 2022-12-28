package command

import (
	"fmt"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

// PrepareDockerLogin executes docker login command if
// SRC_USERNAME/SRC_PASSWORD, DEST_USERNAME/DEST_PASSWORD
// SRC_REGISTRY, DEST_REGISTRY
// environment variables are set
func ProcessDockerLoginEnv() error {
	if u.EnvSourcePassword != "" && u.EnvSourceRegistry != "" {
		err := registry.DockerLogin(
			u.EnvSourceRegistry, u.EnvSourceUsername, u.EnvSourcePassword)
		if err != nil {
			return fmt.Errorf("PrepareDockerLogin: failed to login to %s: %w",
				u.EnvSourceRegistry, err)
		}
	}

	if u.EnvDestPassword != "" && u.EnvDestUsername != "" {
		err := registry.DockerLogin(
			u.EnvDestRegistry, u.EnvDestUsername, u.EnvDestPassword)
		if err != nil {
			return fmt.Errorf("PrepareDockerLogin: failed to login to %s: %w",
				u.EnvDestRegistry, err)
		}
	}

	return nil
}

func DockerLoginRegistry(reg string) error {
	logrus.Infof("Start to login %q", reg)
	username, passwd, err := registry.GetDockerPassword(reg)
	if err != nil {
		logrus.Warnf("Please input password of registry %q manually", reg)
		username, passwd, _ = u.ReadUsernamePasswd()
	}
	return registry.DockerLogin(reg, username, passwd)
}
