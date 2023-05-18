package skopeo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/credential/cache"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

// RunCommandFunc specifies the custom function to run command for skopeo.
// Used for testing purpose!
var RunCommandFunc utils.RunCmdFuncType = nil

const (
	SkopeoName   = "skopeo"
	installGuide = "https://github.com/containers/skopeo/blob/main/install.md"
)

// Installed checks skopeo is installed or not
func Installed() error {
	// ensure skopeo is installed
	p, err := exec.LookPath(SkopeoName)
	if err != nil {
		logrus.Warnf("skopeo not found, please install by refer: %q",
			installGuide)
		return fmt.Errorf("%w", err)
	}
	var buff bytes.Buffer
	cmd := exec.Command(p, "-v")
	cmd.Stdout = &buff
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("'skopeo -v': %w", err)
	}
	logrus.Infof(strings.TrimSpace(buff.String()))

	return nil
}

// Login executes
// 'skopeo login <registry> --username=<user> --password-stdin'
func Login(url, username, password string) error {
	logrus.Debugf("executing 'skopeo login' to %q", url)
	if url == "" {
		url = utils.DockerHubRegistry
	}
	var stdout bytes.Buffer
	args := []string{
		"login",
		url,
		"-u", username,
		"--password-stdin",
	}
	if !config.GetBool("tls-verify") {
		args = append(args, "--tls-verify=false")
	}
	cmd := exec.Command(SkopeoName, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	cmd.Stdin = strings.NewReader(password)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("skopeo login: \n%s\n%w", stdout.String(), err)
	}
	// Login succeed, store username, passwd, registry into cache
	if !cache.Cached(username, password, url) {
		cache.Add(username, password, url)
	}
	return nil
}

// Inspect executs `skopeo inspect ${img}` command
// and return the output if execute successfully
func Inspect(img string, args ...string) (string, error) {
	var execCommandFunc utils.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else {
		execCommandFunc = utils.DefaultRunCommandFunc
	}

	// Inspect the source image info
	param := []string{
		"--insecure-policy", // permissive policy that allows anything
		"inspect",
		img,
	}
	if !config.GetBool("tls-verify") {
		param = append(param, "--tls-verify=false")
	}
	param = append(param, args...)
	logrus.Debugf("running skopeo %v", param)
	var out bytes.Buffer
	if err := execCommandFunc(SkopeoName, nil, &out, param...); err != nil {
		return "", fmt.Errorf("SkopeoInspect %s:\n%w", img, err)
	}

	return out.String(), nil
}

// Copy execute the `skopeo copy ${source} ${destination} args...` cmd.
// You can add custom parameters in `args`
func Copy(src, dst string, args ...string) error {
	var execCommandFunc utils.RunCmdFuncType
	if RunCommandFunc != nil {
		execCommandFunc = RunCommandFunc
	} else {
		execCommandFunc = utils.DefaultRunCommandFunc
	}

	// skopeo copy src dst args...
	param := []string{
		"--insecure-policy", // permissive policy that allows anything
		"copy",
		src,
		dst,
	}
	if !config.GetBool("tls-verify") {
		param = append(param, "--src-tls-verify=false")
		param = append(param, "--dest-tls-verify=false")
	}
	param = append(param, args...)
	logrus.Debugf("running skopeo %v", param)
	var out io.Writer = nil
	if config.GetInt("jobs") == 1 {
		// single thread (worker) mode
		out = os.Stdout
	}

	if err := execCommandFunc(SkopeoName, nil, out, param...); err != nil {
		return fmt.Errorf("SkopeoCopy %s => %s:\n%w", src, dst, err)
	}

	return nil
}
