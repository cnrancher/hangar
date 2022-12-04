package image

import "testing"

func TestImagerInterface(t *testing.T) {
	img := NewImage(&ImageOptions{})
	var imagerer Imagerer = img
	_ = imagerer
}
