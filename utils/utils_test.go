package utils

import (
	"testing"
)

func Test_DefaultRunCommandFunc(t *testing.T) {
	args := []string{"HELLO_WORLD"}
	if out, err := DefaultRunCommandFunc("echo", args...); err != nil || out != "HELLO_WORLD\n" {
		t.Error("DefaultRunCommandFunc 1 failed")
	}
	args = nil
	if out, err := DefaultRunCommandFunc("echo", args...); err != nil || out != "\n" {
		t.Error("DefaultRunCommandFunc 2 failed")
	}
	if out, err := DefaultRunCommandFunc("UNKNOW_CMD", args...); err == nil || out != "" {
		t.Error("DefaultRunCommandFunc 3 failed")
	}
}

func Test_RunCommandStdoutFunc(t *testing.T) {
	args := []string{"HELLO_WORLD"}
	if out, err := RunCommandStdoutFunc("echo", args...); err != nil || out != "" {
		t.Error("DefaultRunCommandFunc 1 failed")
	}
	args = nil
	if out, err := RunCommandStdoutFunc("echo", args...); err != nil || out != "" {
		t.Error("DefaultRunCommandFunc 2 failed")
	}
	if out, err := RunCommandStdoutFunc("UNKNOW_CMD", args...); err == nil || out != "" {
		t.Error("DefaultRunCommandFunc 3 failed")
	}
}

func TestSha256(t *testing.T) {
	s := Sha256Sum("123")
	if s != "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3" {
		t.Errorf("sha256 test failed")
	}
	s = Sha256Sum("<nil>")
	if s != "a9dc16a7a3875d174c1af4f923f261cafc124357f2322493a59ee0d14fcd10db" {
		t.Errorf("sha256 test failed")
	}
}
