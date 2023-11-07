package utils

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	logrus.SetOutput(io.Discard)
}

func Test_Sha256Sum(t *testing.T) {
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

func Test_DecodeBase64(t *testing.T) {
	s := Base64("123")
	if d, e := DecodeBase64(s); e != nil || d != "123" {
		t.Errorf("DecodeBase64 failed %q %v", d, e)
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
// SaveSlice       should be test manually

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

func Test_ReplaceProjectName(t *testing.T) {
	assert.Equal(t, ReplaceProjectName("nginx", ""), "nginx")
	assert.Equal(t, ReplaceProjectName("library/nginx", ""), "nginx")
	assert.Equal(t, ReplaceProjectName("docker.io/nginx", ""), "docker.io/nginx")
	assert.Equal(t, ReplaceProjectName("docker.io/library/nginx", ""), "docker.io/nginx")
	assert.Equal(t, ReplaceProjectName("nginx", "library"), "library/nginx")
	assert.Equal(t, ReplaceProjectName("library/nginx", "another_library"), "another_library/nginx")
	assert.Equal(t, ReplaceProjectName("docker.io/nginx", "library"), "docker.io/library/nginx")
	assert.Equal(t, ReplaceProjectName("docker.io/name/nginx", "library"), "docker.io/library/nginx")
}

// ReadUsernamePasswd should test manually

func Test_SemverCompare(t *testing.T) {
	if res, err := SemverCompare("1.0.0", "1.0.0"); res != 0 || err != nil {
		t.Error("failed:", err, res)
	}
	if res, err := SemverCompare("v1.0.0", "v1.1.0"); res != -1 || err != nil {
		t.Error("failed:", err, res)
	}
	if res, err := SemverCompare("1.1.0", "1.0.0"); res != 1 || err != nil {
		t.Error("failed:", err, res)
	}
	if res, err := SemverCompare("1.0.0-rc", "1.0.0"); res != -1 || err != nil {
		t.Error("failed:", err, res)
	}
	if res, err := SemverCompare("1.0.0-rc1", "1.0.0-rc2"); res != -1 || err != nil {
		t.Error("failed:", err, res)
	}
}

func Test_SemverMajorEqual(t *testing.T) {
	if res := SemverMajorEqual("1.0.0", "1.2.0"); !res {
		t.Error("failed:", res)
	}
	if res := SemverMajorEqual("1.0.0", "1.0.1"); !res {
		t.Error("failed:", res)
	}
	if res := SemverMajorEqual("1.0.0", "1.0.0"); !res {
		t.Error("failed:", res)
	}
	if res := SemverMajorEqual("1.0.0", "2.0.0"); res {
		t.Error("failed:", res)
	}
	if res := SemverMajorEqual("1.0", "2.0"); res {
		t.Error("failed:", res)
	}
}

func Test_SemverMajorMinorEqual(t *testing.T) {
	if res := SemverMajorMinorEqual("1.0.0", "1.2.0"); res {
		t.Error("failed:", res)
	}
	if res := SemverMajorMinorEqual("1.0.0", "1.0.1"); !res {
		t.Error("failed:", res)
	}
	if res := SemverMajorMinorEqual("1.0.0", "1.0.0"); !res {
		t.Error("failed:", res)
	}
	if res := SemverMajorMinorEqual("1.0", "1.0"); !res {
		t.Error("failed:", res)
	}
	if res := SemverMajorMinorEqual("1.0.0", "2.0.0"); res {
		t.Error("failed:", res)
	}
	if res := SemverMajorMinorEqual("1.0", "2.0"); res {
		t.Error("failed:", res)
	}
}

func Test_GetProjectName(t *testing.T) {
	assert.Equal(t, GetProjectName("nginx"), "")
	assert.Equal(t, GetProjectName("docker.io/nginx"), "")
	assert.Equal(t, GetProjectName("library/nginx"), "library")
	assert.Equal(t, GetProjectName("docker.io/library/nginx"), "library")
}

func Test_GetRegistryName(t *testing.T) {
	assert.Equal(t, GetRegistryName("nginx"), "docker.io")
	assert.Equal(t, GetRegistryName("reg.io/nginx"), "reg.io")
	assert.Equal(t, GetRegistryName("library/nginx"), "docker.io")
	assert.Equal(t, GetRegistryName("reg.io/library/nginx"), "reg.io")
}

func Test_GetImageName(t *testing.T) {
	assert.Equal(t, GetImageName("nginx"), "nginx")
	assert.Equal(t, GetImageName("nginx:latest"), "nginx")
	assert.Equal(t, GetImageName("library/nginx"), "nginx")
	assert.Equal(t, GetImageName("library/nginx:latest"), "nginx")
	assert.Equal(t, GetImageName("docker.io/nginx"), "nginx")
	assert.Equal(t, GetImageName("docker.io/nginx:latest"), "nginx")
	assert.Equal(t, GetImageName("docker.io/library/nginx"), "nginx")
	assert.Equal(t, GetImageName("docker.io/library/nginx:latest"), "nginx")
}
