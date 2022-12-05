package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// RunCmdFuncType defines the type of the function to execute command
type RunCmdFuncType func(name string, args ...string) (string, error)

// DefaultRunCommandFunc executes the command and returns the cmd stdout output
func DefaultRunCommandFunc(cmdName string, args ...string) (string, error) {
	// Inspect the source image info
	cmd := exec.Command(cmdName, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s, %w", stdout.String(), err)
	}

	return stdout.String(), nil
}

// RunCommandStdoutFunc executes the command and redirect the output to stdout,
// not recommand to use this func in async mode
func RunCommandStdoutFunc(cmdName string, args ...string) (string, error) {
	// Inspect the source image info
	cmd := exec.Command(cmdName, args...)
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s, %w", stderr.String(), err)
	}

	return "", nil
}

// TODO: handle command execute timeout
