package mirror

import (
	"testing"

	"cnrancher.io/image-tools/registry"
)

func Test_StartSave(t *testing.T) {
	var mirror *Mirror = nil
	var err error

	if err = mirror.StartSave(); err == nil {
		t.Error("StartSave failed")
	}

	mirror = NewMirror(&MirrorOptions{
		Mode: 0,
	})
	if err = mirror.StartSave(); err == nil {
		t.Error("StartSave failed")
	}
	mirror = NewMirror(&MirrorOptions{
		Source:      "",
		Destination: "",
		Tag:         "",
		ArchList:    []string{"amd64", "arm64"},
		Directory:   "",
		Mode:        MODE_SAVE,
	})
	// mirror.AppendImage(image.NewImage(&image.ImageOptions{}))

	// fake skopeo inspect / skopeo copy function
	registry.RunCommandFunc = func(a string, p ...string) (string, error) {
		out, err := testFs.ReadFile(TestS2V2ListFileName)
		return string(out), err
	}
	if err = mirror.StartSave(); err != nil {
		t.Error(err)
	}

	if mirror.Saved() != 2 {
		t.Error("Saved failed")
	}
}
