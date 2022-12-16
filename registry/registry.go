package registry

import (
	"fmt"
	"os/exec"
	"strings"

	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

// RunCommandFunc specifies the custom function to run command for registry.
//
// Only used for testing purpose!
var RunCommandFunc u.RunCmdFuncType = nil

// SelfCheck checks the registry related commands is installed or not
func SelfCheckSkopeo() error {
	// ensure skopeo is installed
	skopeoPath, err := EnsureSkopeoInstalled("")
	if err != nil {
		return fmt.Errorf("SelfCheckSkopeo: %w", err)
	}
	if strings.HasPrefix(skopeoPath, "/") {
		logrus.Debug("skopeo is a system application")
	} else {
		logrus.Debug("skopeo is at current folder as a executable binary file")
	}

	return nil
}

func SelfCheckBuildX() error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("SelfCheckBuildX: %w", u.ErrDockerNotFound)
	}

	var execCommandFunc u.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else {
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// ensure docker-buildx is installed
	if err = execCommandFunc(dockerPath, nil, nil, "buildx"); err != nil {
		if strings.Contains(err.Error(), "is not a docker command") {
			return fmt.Errorf("SelfCheckBuildX: %w", u.ErrDockerBuildxNotFound)
		}
	}

	return nil
}

func SelfCheckDocker() error {
	// check docker
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("SelfCheckDocker: %w", u.ErrDockerNotFound)
	}

	return nil
}
