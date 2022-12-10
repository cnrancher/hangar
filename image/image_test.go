package image

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetOutput(ioutil.Discard)
}

func TestImagerInterface(t *testing.T) {
	img := NewImage(&ImageOptions{})
	var imager Imager = img
	_ = imager
}

func Test_NewImage(t *testing.T) {
	var imager Imager = NewImage(&ImageOptions{
		Source:              "docker.io/example",
		Destination:         "private.io/library/example",
		Tag:                 "v1.0.0",
		Arch:                "arm64",
		Variant:             "v8",
		OS:                  "linux",
		Digest:              "sha256:" + u.Sha256Sum("ABC"),
		Directory:           "test",
		SavedFolder:         u.Sha256Sum("library/hello-world"),
		SourceSchemaVersion: 2,
		SourceMediaType:     u.MediaTypeManifestV2,
		MID:                 fmt.Sprintf("%02d", 1),
	})

	if s := imager.Source(); s != "docker.io/example" {
		t.Error("Source failed")
	}
	if d := imager.Destination(); d != "private.io/library/example" {
		t.Error("Destination failed")
	}
	if a := imager.Arch(); a != "arm64" {
		t.Error("Arch failed")
	}
	if o := imager.OS(); o != "linux" {
		t.Error("OS failed")
	}
	if d := imager.Digest(); d != "sha256:"+u.Sha256Sum("ABC") {
		t.Error("Digest failed")
	}
	imager.SetDigest("sha256:" + u.Sha256Sum("XYZ"))
	if d := imager.Digest(); d != "sha256:"+u.Sha256Sum("XYZ") {
		t.Error("SetDigest failed")
	}
	if imager.Directory() != "test" {
		t.Error("Directory failed")
	}
	if imager.SavedFolder() != u.Sha256Sum("library/hello-world") {
		t.Error("SavedFolder failed")
	}
	imager.SetID("01")
	if i := imager.ID(); i != "01" {
		t.Error("SetID failed")
	}
	imager.SetID("02")
	if i := imager.ID(); i != "02" {
		t.Error("SetID failed")
	}
	imager.SetMID("03")
	if imager.MID() != "03" {
		t.Error("SetMID failed")
	}
}

func Test_Copy(t *testing.T) {
	// nil pointer
	var imageNil *Image = nil
	var imagerNil Imager = imageNil
	if err := imagerNil.Copy(); err == nil {
		t.Error("Copy failed")
	}

	var imagerEmpty Imager = NewImage(&ImageOptions{})
	if err := imagerEmpty.Copy(); err == nil {
		t.Error("Copy failed")
	}

	var imagerV2 Imager = NewImage(&ImageOptions{
		Source:              "docker.io/example",
		Destination:         "private.io/library/example",
		Tag:                 "v1.0.0",
		Arch:                "arm64",
		Variant:             "v8",
		OS:                  "linux",
		Digest:              "sha256:" + u.Sha256Sum("ABC"),
		SourceSchemaVersion: 2,
		SourceMediaType:     u.MediaTypeManifestV2,
		MID:                 fmt.Sprintf("%02d", 1),
	})

	// fake skopeo copy, skopeo inspect func
	// this function will make source digest equals to dest digest
	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		return "FAKE_OUTPUT\n", nil
	}
	if err := imagerV2.Copy(); err != nil {
		t.Error(err.Error())
	}
	if !imagerV2.Copied() {
		t.Error("Copy failed")
	}

	var imagerV1 Imager = NewImage(&ImageOptions{
		Source:              "docker.io/example",
		Destination:         "private.io/library/example",
		Tag:                 "v1.0.0",
		Arch:                "arm64",
		Variant:             "v8",
		OS:                  "linux",
		Digest:              "sha256:" + u.Sha256Sum("ABC"),
		SourceSchemaVersion: 1,
		SourceMediaType:     u.MediaTypeManifestV2,
		MID:                 fmt.Sprintf("%02d", 1),
	})
	if err := imagerV1.Copy(); err != nil {
		t.Error(err.Error())
	}
	if !imagerV1.Copied() {
		t.Error("Copy failed")
	}

	// return random output, this will make source digest not equal to dest
	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		// sleep 100ms
		time.Sleep(time.Microsecond * 100)
		return time.Now().Format(time.StampNano) + "\n", nil
	}

	var imagerListV2 Imager = NewImage(&ImageOptions{
		Source:              "docker.io/example",
		Destination:         "private.io/library/example",
		Tag:                 "v1.0.0",
		Arch:                "arm64",
		Variant:             "v8",
		OS:                  "linux",
		Digest:              "sha256:" + u.Sha256Sum("ABC"),
		SourceSchemaVersion: 2,
		SourceMediaType:     u.MediaTypeManifestListV2,
		MID:                 fmt.Sprintf("%02d", 1),
	})
	if err := imagerListV2.Copy(); err != nil {
		t.Error(err.Error())
	}
	if !imagerListV2.Copied() {
		t.Error("Copy failed")
	}
	registry.RunCommandFunc = nil
}

func Test_CopiedTag(t *testing.T) {
	if CopiedTag("1", "linux", "amd64", "") != "1-amd64" {
		t.Error("CopiedTag failed")
	}
	if CopiedTag("1", "linux", "arm64", "v8") != "1-arm64" {
		t.Error("CopiedTag failed")
	}
	if CopiedTag("1", "linux", "arm", "v7") != "1-armv7" {
		t.Error("CopiedTag failed")
	}
	if CopiedTag("1", "linux", "s390x", "") != "1-s390x" {
		t.Error("CopiedTag failed")
	}
	if CopiedTag("1", "darwin", "amd64", "") != "1-darwin-amd64" {
		t.Error("CopiedTag failed")
	}
	if CopiedTag("1", "darwin", "arm64", "v8") != "1-darwin-arm64" {
		t.Error("CopiedTag failed")
	}
	if CopiedTag("1", "windows", "amd64", "") != "1-windows-amd64" {
		t.Error("CopiedTag failed")
	}
	if CopiedTag("1", "windows", "arm64", "v8") != "1-windows-arm64" {
		t.Error("CopiedTag failed")
	}
}

func Test_Load(t *testing.T) {
	img := Image{
		source:      ".", // source is a directory
		destination: "priv.io/library/nginx",
	}
	// fake skopeo copy function
	registry.RunCommandFunc = func(a string, p ...string) (string, error) {
		return "", nil
	}
	if err := img.Load(); err != nil {
		t.Fatal(err)
	}
	if !img.Loaded() {
		t.Error("load failed")
	}
	registry.RunCommandFunc = nil
}

func Test_Save(t *testing.T) {
	img := Image{
		source:      "priv.io/library/nginx",
		destination: ".", // dest is a directory
		directory:   ".",
	}
	// fake skopeo copy function
	registry.RunCommandFunc = func(a string, p ...string) (string, error) {
		return "", nil
	}
	if err := img.Save(); err != nil {
		t.Fatal(err)
	}
	if !img.Saved() {
		t.Error("load failed")
	}
	registry.RunCommandFunc = nil
}
