package mirror

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestMirrorerInterface(t *testing.T) {
	mirror := NewMirror(&MirrorOptions{})
	var mirrorer Mirrorer = mirror
	_ = mirrorer
}

func TestConstructureRegistry(t *testing.T) {
	s := constructRegistry("nginx", "")
	if s != "docker.io/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = constructRegistry("docker.io/nginx", "")
	if s != "docker.io/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = constructRegistry("localhost/nginx", "")
	if s != "localhost/nginx" {
		t.Error("value should be 'localhost/nginx'")
	}

	s = constructRegistry("custom.io/nginx", "")
	if s != "custom.io/nginx" {
		t.Error("value should be 'custom.io/nginx'")
	}

	dstReg := "private.io"

	s = constructRegistry("nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}

	s = constructRegistry("docker.io/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}

	s = constructRegistry("localhost/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}

	s = constructRegistry("custom.io/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}
}
