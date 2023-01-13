package registry

import (
	"strings"
	"testing"
)

func Test_CreateHarborProject(t *testing.T) {
	var url string
	// EDIT THIS LINE MANUALLY
	// url = "https://harbor2.private.io/api/v2.0/projects"

	if url == "" {
		return
	}
	if err := CreateHarborProject("name", url, "", ""); err != nil {
		t.Error(err)
	}
}

func Test_GetDockerPasswdByConfig(t *testing.T) {
	// auth is base64 encoded username:password
	config := `
	{
		"auths": {
			"test.io": {
				"auth": "dXNlcjpwYXNzd2Q="
			},
			"https://index.docker.io/v1/": {
				"auth": "dXNlcjpwYXNzd2Q="
			}
		}
	}`
	// the config contains credsStore should be tested manually (macOS)

	user, passwd, err := GetDockerPasswdByConfig(
		"docker.io", strings.NewReader(config))
	if err != nil {
		t.Error("GetDockerPasswdByConfig failed", err)
	}
	if user != "user" {
		t.Error("GetDockerPasswdByConfig failed")
	}
	if passwd != "passwd" {
		t.Error("GetDockerPasswdByConfig failed")
	}
	user, passwd, err = GetDockerPasswdByConfig(
		"test.io", strings.NewReader(config))
	if err != nil {
		t.Error("GetDockerPasswdFromConfig failed", err)
	}
	if user != "user" {
		t.Error("GetDockerPasswdFromConfig failed")
	}
	if passwd != "passwd" {
		t.Error("GetDockerPasswdFromConfig failed")
	}
}
