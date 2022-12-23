package registry

import (
	"bytes"
	"fmt"
	"io"
	"os"
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

	logrus.Warnf("skopeo not found, please install skopeo manually: %s",
		skopeoInsGuideURL)
	return "", u.ErrSkopeoNotFound
}

// InspectRaw function executs `skopeo inspect ${img}` command
// and return the output if execute successfully
func SkopeoInspect(img string, args ...string) (string, error) {
	// Ensure skopeo installed
	skopeo, err := EnsureSkopeoInstalled("")
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
	param := []string{
		"--insecure-policy", // permissive policy that allows anything
		"inspect",
		img,
		"--tls-verify=false",
	}
	param = append(param, args...)
	logrus.Debugf("Running skopeo %v", param)
	var out bytes.Buffer
	if err := execCommandFunc(skopeo, nil, &out, param...); err != nil {
		return "", fmt.Errorf("SkopeoInspect %s:\n%w", img, err)
	}

	return out.String(), nil
}

// SkopeoCopy execute the `skopeo copy ${source} ${destination} args...` cmd
// You can add custom parameters in `args`
func SkopeoCopy(src, dst string, args ...string) error {
	skopeo, err := EnsureSkopeoInstalled("")
	if err != nil {
		return fmt.Errorf("unable to locate skopeo path: %w", err)
	}

	var execCommandFunc u.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else {
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// skopeo copy src dst args...
	params := []string{
		"--insecure-policy", // permissive policy that allows anything
		"copy",
		src,
		dst,
		"--src-tls-verify=false",
		"--dest-tls-verify=false",
	}
	params = append(params, args...)
	logrus.Debugf("Running skopeo %v", params)
	var out io.Writer = nil
	if u.WorkerNum == 1 {
		// single thread (worker) mode
		out = os.Stdout
	}

	if err = execCommandFunc(skopeo, nil, out, params...); err != nil {
		return fmt.Errorf("SkopeoCopy %s => %s:\n%w", src, dst, err)
	}

	return nil
}
