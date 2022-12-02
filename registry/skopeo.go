package registry

import (
	"bytes"
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
func EnsureSkopeoInstalled(path string) (string, error) {
	if path, err := exec.LookPath("skopeo"); err == nil {
		logrus.Debugf("found skopeo installed at: %v", path)
		return path, nil
	}

	if _, err := os.Stat(filepath.Join(path, "skopeo")); err == nil {
		logrus.Debug("skopeo already downloaded.")
		return filepath.Join(path, "skopeo"), nil
	}

	if runtime.GOOS != "linux" {
		logrus.Warnf("Your OS is %s, please install skopeo manually: %s",
			runtime.GOOS, skopeoInsGuideURL)
		return "", fmt.Errorf("unsupported system: %v", runtime.GOOS)
	}

	logrus.Info("skopeo not found, trying to download binary file.")
	out, err := os.OpenFile(
		filepath.Join(path, "skopeo"),
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

	return filepath.Join(path, "skopeo"), nil
}

// InspectRaw function executs `skopeo inspect --raw ${img}` command
// and return the output if execute successfully
func SkopeoInspect(img string, extraArgs ...string) (*bytes.Buffer, error) {
	logrus.Debug("Running skopeo inspect...")
	// Ensure skopeo installed
	skopeoPath, err := EnsureSkopeoInstalled("")
	if err != nil {
		return nil, fmt.Errorf("unable to locate skopeo path: %w", err)
	}

	// Inspect the source image info
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var args []string
	args = append(args, "inspect", img)
	args = append(args, extraArgs...)
	cmd := exec.Command(skopeoPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("SkopeoInspect: \n%s\n%w",
			stderr.String(), err)
	}

	return &stdout, nil
}

// SkopeoCopyArchOS
// `skopeo copy --override-arch={ARCH} --override-os={OS}`;
// You can add --override-variant={VARIANT} in `extraArgs`
// the os parameter can be set to empty string,
// extraArgs can be nil
func SkopeoCopyArchOS(arch, osType, source, dest string, extraArgs ...string) error {
	logrus.Debug("Running skopeo copy...")
	skopeoPath, err := EnsureSkopeoInstalled("")
	if err != nil {
		return fmt.Errorf("unable to locate skopeo path: %w", err)
	}

	// Inspect the source image info
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var args []string
	args = append(args, "copy", "--override-arch="+arch)
	if osType != "" {
		args = append(args, "--override-os="+osType)
	}
	args = append(args, extraArgs...)
	cmd := exec.Command(skopeoPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("SkopeoCopyArchOS: \n%s\n%w",
			stderr.String(), err)
	}

	return nil
}
