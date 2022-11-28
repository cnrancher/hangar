package mirror

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestMirrorImage(t *testing.T) {
	archList := []string{"arm64", "amd64"}

	err := mirrorImage("nginx", "example/nginx", "1.22", archList)
	if err != nil {
		t.Error(err.Error())
	}
}
