package utils

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

// RunCmdFuncType defines the type of the function to execute command
type RunCmdFuncType func(name string, in io.Reader, out io.Writer, args ...string) error

// DefaultRunCommandFunc executes the command and specifies the stdin
// and stdout in parameters
func DefaultRunCommandFunc(p string, i io.Reader, o io.Writer, args ...string) error {
	// Inspect the source image info
	cmd := exec.Command(p, args...)
	var stderr bytes.Buffer
	cmd.Stdin = i
	cmd.Stdout = o
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s, %w", stderr.String(), err)
	}

	return nil
}

// TODO: handle command execute timeout
