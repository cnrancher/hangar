package mirror

import (
	"strings"
	"testing"

	"cnrancher.io/image-tools/image"
)

func Test_NewSavedListTemplate(t *testing.T) {
	mT := NewSavedListTemplate()
	if len(mT.List) != 0 {
		t.Error("NewSavedListTemplate failed")
	}
	mT.Append(&SavedMirrorTemplate{})
	if len(mT.List) != 1 {
		t.Error("NewSavedListTemplate failed")
	}
}

func Test_GetSavedImageTemplate(t *testing.T) {
	mirror := NewMirror(&MirrorOptions{
		Mode: MODE_MIRROR,
	})
	mT := mirror.GetSavedImageTemplate()
	if mT != nil {
		t.Error("GetSavedImageTemplate failed")
	}

	mirror = NewMirror(&MirrorOptions{
		Source:      "docker.io/nginx",
		Destination: "priv.io/nginx",
		Tag:         "1",
		Directory:   "",
		ArchList:    []string{"amd64", "arm64"},
		Mode:        MODE_SAVE,
	})
	// mirror does not have images, return nil
	mT = mirror.GetSavedImageTemplate()
	if mT != nil {
		t.Error("GetSavedImageTemplate failed")
	}

	mirror.AppendImage(image.NewImage(&image.ImageOptions{
		Arch: "amd64",
	}))

	mT = mirror.GetSavedImageTemplate()
	if mT == nil {
		t.Fatal("GetSavedImageTemplate failed")
	}
	if mT.Images == nil {
		t.Error("GetSavedImageTemplate failed")
	}
	if len(mT.Images) != 1 {
		t.Error("GetSavedImageTemplate failed")
	}
	mirror.AppendImage(image.NewImage(&image.ImageOptions{
		Arch:    "arm",
		Variant: "v7",
	}))
	mirror.AppendImage(image.NewImage(&image.ImageOptions{
		Arch:    "arm",
		Variant: "v5",
	}))
	mT = mirror.GetSavedImageTemplate()
	if len(mT.Images) != 3 {
		t.Error("GetSavedImageTemplate failed")
	}
}

func Test_LoadSavedTemplates(t *testing.T) {
	mirrorer, err := LoadSavedTemplates("test/", "custom.io")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mirrorer {
		if m.Source() != "harbor2.hxstarrys.me/library/nginx" {
			t.Error("Source failed")
		}
		if !m.HasArch("amd64") || !m.HasArch("arm64") {
			t.Error("HasArch failed")
		}
		if m.Mode() != MODE_LOAD {
			t.Error("Mode failed")
		}
		// the directory is converted to absolute path
		if !strings.HasSuffix(m.Directory(), "test") {
			t.Error("Directory failed")
		}
		if m.ImageNum() != 2 {
			t.Error("ImageNum failed")
		}
	}
}
