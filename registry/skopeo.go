package registry

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

const (
	skopeoAmd64URL    = "https://starry-public-files.s3.ap-northeast-1.amazonaws.com/skopeo/amd64/1.9.3/skopeo"
	skopeoArm64URL    = "https://starry-public-files.s3.ap-northeast-1.amazonaws.com/skopeo/arm64/1.9.3/skopeo"
	skopeoInsGuideURL = "https://github.com/containers/skopeo/blob/main/install.md"
)

// EnsureSkopeoInstalled ensures the skopeo command is installed.
// If the skopeo is not instqlled, download the binary to current dir.
func EnsureSkopeoInstalled(installPath string) (string, error) {
	var path string
	var err error
	if path, err = exec.LookPath("skopeo"); err == nil {
		logrus.Debugf("Found skopeo installed at: %v", path)
		return path, nil
	}

	if _, err = os.Stat(filepath.Join(installPath, "skopeo")); err == nil {
		logrus.Debug("skopeo already downloaded.")
		return filepath.Join(installPath, "skopeo"), nil
	}

	if runtime.GOOS != "linux" {
		logrus.Warnf("Your OS is %s, please install skopeo manually: %s",
			runtime.GOOS, skopeoInsGuideURL)
		return "", fmt.Errorf("unsupported system: %v", runtime.GOOS)
	}

	logrus.Info("skopeo not found, trying to download binary file.")
	out, err := os.OpenFile(
		filepath.Join(installPath, "skopeo"),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0755)
	if err != nil {
		return "", fmt.Errorf("InstallSkopeo: %w", err)
	}
	defer out.Close()

	var resp *http.Response
	switch runtime.GOARCH {
	case "amd64":
		resp, err = http.Get(skopeoAmd64URL)
	case "arm64":
		resp, err = http.Get(skopeoArm64URL)
	default:
		logrus.Warnf("skopeo not found, please install manually: %s",
			skopeoInsGuideURL)
		return "", u.ErrSkopeoNotFound
	}
	if err != nil {
		return "", fmt.Errorf("InstallSkopeo: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("InstallSkopeo: %w", err)
	}

	return filepath.Join(installPath, "skopeo"), nil
}

// InspectRaw function executs `skopeo inspect ${img}` command
// and return the output if execute successfully
func SkopeoInspect(img string, args ...string) (string, error) {
	logrus.Debugf("Running skopeo inspect [%s] %v", img, args)
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
	logrus.Debugf("Running skopeo copy src[%s] dst[%s] %v", src, dst, args)
	skopeoPath, err := EnsureSkopeoInstalled("")
	if err != nil {
		return fmt.Errorf("unable to locate skopeo path: %w", err)
	}

	var execCommandFunc u.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else if u.MirrorerJobNum == 1 {
		// if not async mode, set command output to stdout
		execCommandFunc = u.RunCommandStdoutFunc
	} else {
		// if in async mode, set command output to io.Writer instead of stdout
		execCommandFunc = u.DefaultRunCommandFunc
	}

	// skopeo copy src dst args...
	params := []string{"copy", src, dst}
	params = append(params, args...)

	stdout, err := execCommandFunc(skopeoPath, params...)
	if err != nil {
		return fmt.Errorf("SkopeoCopy %s => %s:\n%w", src, dst, err)
	}
	fmt.Print(stdout)

	return nil
}
