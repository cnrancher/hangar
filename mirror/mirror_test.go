package mirror

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

// func TestMirrorImage(t *testing.T) {
// 	archList := []string{"arm64", "amd64"}

// 	err := mirrorImage("nginx", "example/nginx", "1.22", archList)
// 	if err != nil {
// 		t.Error(err.Error())
// 	}
// }

func TestConstructureRegistry(t *testing.T) {
	s := constructureRegistry("nginx", "")
	if s != "docker.io/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = constructureRegistry("docker.io/nginx", "")
	if s != "docker.io/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = constructureRegistry("localhost/nginx", "")
	if s != "localhost/nginx" {
		t.Error("value should be 'localhost/nginx'")
	}

	s = constructureRegistry("custom.io/nginx", "")
	if s != "custom.io/nginx" {
		t.Error("value should be 'custom.io/nginx'")
	}

	dstReg := "private.io"

	s = constructureRegistry("nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = constructureRegistry("docker.io/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = constructureRegistry("localhost/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Error("value should be 'localhost/nginx'")
	}

	s = constructureRegistry("custom.io/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Error("value should be 'custom.io/nginx'")
	}
}
