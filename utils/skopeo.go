package utils

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

const (
	skopeoAmd64URL = "https://starry-public-files.s3.ap-northeast-1.amazonaws.com/skopeo/amd64/1.9.3/skopeo"
	skopeoArm64URL = "https://starry-public-files.s3.ap-northeast-1.amazonaws.com/skopeo/arm64/1.9.3/skopeo"
)

// EnsureSkopeoInstalled ensures the skopeo command is installed.
// If the skopeo is not instqlled, download the binary to current dir.
func EnsureSkopeoInstalled(path string) (string, error) {
	if path, err := exec.LookPath("skopeo"); err == nil {
		logrus.Infof("found skopeo installed at: %v", path)
		return path, nil
	}

	if _, err := os.Stat(filepath.Join(path, "skopeo")); err == nil {
		logrus.Debug("skopeo already downloaded.")
		return filepath.Join(path, "skopeo"), nil
	}

	if runtime.GOOS != "linux" {
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
		return "", errors.New("unsupported arch")
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
