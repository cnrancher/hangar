package registry

import (
	"fmt"
	"os/exec"

	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

const (
	skopeoInsGuideURL = "https://github.com/containers/skopeo/blob/main/install.md"
)

// EnsureSkopeoInstalled ensures the skopeo command is installed.
func EnsureSkopeoInstalled(installPath string) (string, error) {
	var path string
	var err error
	if path, err = exec.LookPath("skopeo"); err == nil {
		logrus.Debugf("Found skopeo installed at: %v", path)
		return path, nil
	}

	logrus.Warnf("skopeo not found, lease install skopeo manually: %s",
		skopeoInsGuideURL)
	return "", u.ErrSkopeoNotFound
}

// InspectRaw function executs `skopeo inspect ${img}` command
// and return the output if execute successfully
func SkopeoInspect(img string, args ...string) (string, error) {
	// Ensure skopeo installed
	skopeoPath, err := EnsureSkopeoInstalled("")
	if err != nil {
		return "", fmt.Errorf("unable to locate skopeo path: %w", err)
	}

	var execCommandFunc u.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else {
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// Inspect the source image info
	param := []string{"inspect", img}
	// default policy: permissive policy that allows anything.
	args = append(
		args,
		"--insecure-policy",
		"--tls-verify=false",
	)
	logrus.Debugf("Running skopeo inspect [%s] %v", img, args)
	param = append(param, args...)

	out, err := execCommandFunc(skopeoPath, param...)
	if err != nil {
		return "", fmt.Errorf("SkopeoInspect %s:\n%w", img, err)
	}

	return out, nil
}

// SkopeoCopy execute the `skopeo copy ${source} ${destination} args...` cmd
// You can add custom parameters in `args`
func SkopeoCopy(src, dst string, args ...string) error {
	skopeoPath, err := EnsureSkopeoInstalled("")
	if err != nil {
		return fmt.Errorf("unable to locate skopeo path: %w", err)
	}

	var execCommandFunc u.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else if u.WorkerNum == 1 {
		// if not async mode, set command output to stdout
		execCommandFunc = u.RunCommandStdoutFunc
	} else {
		// if in async mode, set command output to io.Writer instead of stdout
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// skopeo copy src dst args...
	params := []string{"copy", src, dst}
	// default policy: permissive policy that allows anything.
	args = append(
		args,
		"--insecure-policy",
		"--src-tls-verify=false",
		"--dest-tls-verify=false",
	)
	params = append(params, args...)
	logrus.Debugf("Running skopeo copy src[%s] dst[%s] %v", src, dst, args)

	if _, err = execCommandFunc(skopeoPath, params...); err != nil {
		return fmt.Errorf("SkopeoCopy %s => %s:\n%w", src, dst, err)
	}

	return nil
}
