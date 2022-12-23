package registry

import (
	"bytes"
	"fmt"
	"io"
	"os"

	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

// InspectRaw function executs `skopeo inspect ${img}` command
// and return the output if execute successfully
func SkopeoInspect(img string, args ...string) (string, error) {
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
	if err := execCommandFunc(SkopeoPath, nil, &out, param...); err != nil {
		return "", fmt.Errorf("SkopeoInspect %s:\n%w", img, err)
	}

	return out.String(), nil
}

// SkopeoCopy execute the `skopeo copy ${source} ${destination} args...` cmd
// You can add custom parameters in `args`
func SkopeoCopy(src, dst string, args ...string) error {
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

	if err := execCommandFunc(SkopeoPath, nil, out, params...); err != nil {
		return fmt.Errorf("SkopeoCopy %s => %s:\n%w", src, dst, err)
	}

	return nil
}
