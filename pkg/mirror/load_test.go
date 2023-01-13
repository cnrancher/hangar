package mirror

import (
	"io"
	"testing"

	"github.com/cnrancher/image-tools/pkg/image"
	"github.com/cnrancher/image-tools/pkg/registry"
)

func Test_StartLoad(t *testing.T) {
	var mirror *Mirror = nil
	var err error
	if err = mirror.StartLoad(); err == nil {
		t.Error("StartLoad failed")
	}

	mirror = NewMirror(&MirrorOptions{
		Directory: "",
		Mode:      MODE_LOAD,
	})
	if err = mirror.StartLoad(); err == nil {
		t.Error("StartLoad failed")
	}

	mirror = NewMirror(&MirrorOptions{
		Directory: "test",
		Mode:      0,
	})
	if err = mirror.StartLoad(); err == nil {
		t.Error("StartLoad failed")
	}

	mirror = NewMirror(&MirrorOptions{
		Directory: "test",
		Mode:      MODE_LOAD,
	})

	mirror.AppendImage(image.NewImage(&image.ImageOptions{
		Source:      "./test", // source is the local directory of the image
		Destination: "custom.io/library/nginx",
		Tag:         "1",
		Directory:   "test",
		SavedFolder: ".",
		Arch:        "amd64",
	}))
	mirror.AppendImage(image.NewImage(&image.ImageOptions{
		Source:      "./test",
		Destination: "custom.io/library/nginx",
		Tag:         "1",
		Directory:   "test",
		SavedFolder: ".",
		Arch:        "arm64",
	}))
	if mirror.Loaded() != 0 {
		t.Error("Loaded failed")
	}

	// fake skopeo copy function
	fake := func(a string, in io.Reader, out io.Writer, p ...string) error {
		return nil
	}
	registry.RunCommandFunc = fake
	if err = mirror.StartLoad(); err != nil {
		t.Fatal(err)
	}
	if mirror.Loaded() != 2 {
		t.Error("Loaded failed")
	}
}
