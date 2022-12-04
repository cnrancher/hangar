package image

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestImagerInterface(t *testing.T) {
	img := NewImage(&ImageOptions{})
	var imagerer Imagerer = img
	_ = imagerer
}
