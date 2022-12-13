package utils

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetOutput(io.Discard)
}

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

func Test_Sha256sum(t *testing.T) {
	s := Sha256Sum("123")
	if s != "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3" {
		t.Errorf("sha256 test failed")
	}
	s = Sha256Sum("<nil>")
	if s != "a9dc16a7a3875d174c1af4f923f261cafc124357f2322493a59ee0d14fcd10db" {
		t.Errorf("sha256 test failed")
	}
}

func Test_Base64(t *testing.T) {
	s := Base64("123")
	if s != "MTIz" {
		t.Error("base64 test failed")
	}
	s = Base64("Username:Password")
	if s != "VXNlcm5hbWU6UGFzc3dvcmQ=" {
		t.Error("base64 test failed")
	}
}

// func AppendFileLine should be test manually

func Test_IsDirEmpty(t *testing.T) {
	var (
		ok  bool
		err error
	)
	// non-exist folder should return true
	if ok, err = IsDirEmpty("UNKNOW_FOLDER"); !ok || err != nil {
		t.Error("IsDirEmpty failed")
	}
	// current dir is not empty, should return false
	if ok, err = IsDirEmpty("."); ok || err != nil {
		t.Error("IsDirEmpty failed")
	}
}

func Test_GetAbsPath(t *testing.T) {
	var (
		dir string
		err error
	)
	currentDir, _ := os.Getwd()
	// when the parameter is empty string, the return value should be
	// the current absolute dir
	if dir, err = GetAbsPath(""); !strings.HasPrefix(dir, currentDir) || dir != currentDir || err != nil {
		t.Error("GetAbsPath failed")
	}
	if dir, err = GetAbsPath("test"); !strings.HasPrefix(dir, currentDir) || !strings.HasSuffix(dir, "test") || err != nil {
		t.Error("GetAbsPath failed")
	}
	if dir, err = GetAbsPath("/bin/cat"); dir != "/bin/cat" || err != nil {
		t.Error("GetAbsPath failed")
	}
}

// EnsureDirExists should be test manually
// DeleteIfExist   should be test manually
// SaveJson        should be test manually

func Test_CheckWorkerNum(t *testing.T) {
	var num int = 1
	CheckWorkerNum(true, &num)
	if num != 1 {
		t.Error("CheckWorkerNum failed")
	}
	num = 2
	CheckWorkerNum(true, &num)
	if num != 1 {
		t.Error("CheckWorkerNum failed")
	}
	num = 1
	CheckWorkerNum(false, &num)
	if num != 1 {
		t.Error("CheckWorkerNum failed")
	}
	num = 0
	CheckWorkerNum(false, &num)
	if num != 1 {
		t.Error("CheckWorkerNum failed")
	}
	num = 100
	CheckWorkerNum(false, &num)
	if num != 20 {
		t.Error("CheckWorkerNum failed")
	}
}

func Test_ConstructRegistry(t *testing.T) {
	s := ConstructRegistry("nginx", "")
	if s != "docker.io/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = ConstructRegistry("docker.io/nginx", "")
	if s != "docker.io/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = ConstructRegistry("localhost/nginx", "")
	if s != "localhost/nginx" {
		t.Error("value should be 'localhost/nginx'")
	}

	s = ConstructRegistry("custom.io/nginx", "")
	if s != "custom.io/nginx" {
		t.Error("value should be 'custom.io/nginx'")
	}

	dstReg := "private.io"

	s = ConstructRegistry("nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}

	s = ConstructRegistry("docker.io/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}

	s = ConstructRegistry("localhost/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}

	s = ConstructRegistry("custom.io/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}
}

// ReadUsernamePasswd should test manually
