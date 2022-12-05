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
func SelfCheck() error {
	// check docker
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("SelfCheck: %w", u.ErrDockerNotFound)
	}

	var execCommandFunc u.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else {
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// ensure docker-buildx is installed
	checkBuildxInstalledParam := []string{"buildx", "--help"}

	out, err := execCommandFunc(dockerPath, checkBuildxInstalledParam...)
	if err != nil {
		if strings.Contains(out, "is not a docker command") {
			return fmt.Errorf("DockerBuildx: %w", u.ErrDockerBuildxNotFound)
		}
	}

	// ensure skopeo is installed
	skopeoPath, err := EnsureSkopeoInstalled("")
	if err != nil {
		return fmt.Errorf("EnsureSkopeoInstalled: %w", err)
	}
	if strings.HasPrefix(skopeoPath, "/") {
		logrus.Debug("skopeo is a system application")
	} else {
		logrus.Debug("skopeo is at current folder as a executable binary file")
	}

	return nil
}
