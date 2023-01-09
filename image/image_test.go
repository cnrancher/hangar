package image

import (
	"io"
	"testing"
	"time"

	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/containers/image/v5/manifest"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetOutput(io.Discard)
	u.WorkerNum = 2
}

func Test_NewImage(t *testing.T) {
	var image *Image = NewImage(&ImageOptions{
		Source:          "docker.io/example",
		Destination:     "private.io/library/example",
		Tag:             "v1.0.0",
		Arch:            "arm64",
		Variant:         "v8",
		OS:              "linux",
		Digest:          "sha256:" + u.Sha256Sum("ABC"),
		Directory:       "test",
		SavedFolder:     u.Sha256Sum("library/hello-world"),
		SourceMediaType: manifest.DockerV2Schema2MediaType,
		MID:             1,
	})

	if s := image.Source; s != "docker.io/example" {
		t.Error("Source failed")
	}
	if d := image.Destination; d != "private.io/library/example" {
		t.Error("Destination failed")
	}
	if a := image.Arch; a != "arm64" {
		t.Error("Arch failed")
	}
	if o := image.OS; o != "linux" {
		t.Error("OS failed")
	}
	if d := image.Digest; d != "sha256:"+u.Sha256Sum("ABC") {
		t.Error("Digest failed")
	}
	image.Digest = "sha256:" + u.Sha256Sum("XYZ")
	if image.Digest != "sha256:"+u.Sha256Sum("XYZ") {
		t.Error("SetDigest failed")
	}
	if image.Directory != "test" {
		t.Error("Directory failed")
	}
	if image.SavedFolder != u.Sha256Sum("library/hello-world") {
		t.Error("SavedFolder failed")
	}
}

func Test_Copy(t *testing.T) {
	// nil pointer
	var imageNil *Image = nil
	if err := imageNil.Copy(); err == nil {
		t.Error("Copy failed")
	}

	var imageEmpty *Image = NewImage(&ImageOptions{})
	if err := imageEmpty.Copy(); err == nil {
		t.Error("Copy failed")
	}

	var imageV2 *Image = NewImage(&ImageOptions{
		Source:          "docker.io/example",
		Destination:     "private.io/library/example",
		Tag:             "v1.0.0",
		Arch:            "arm64",
		Variant:         "v8",
		OS:              "linux",
		Digest:          "sha256:" + u.Sha256Sum("ABC"),
		SourceMediaType: manifest.DockerV2Schema2MediaType,
		MID:             1,
	})

	// fake skopeo copy, skopeo inspect func
	// this function will make source digest equals to dest digest
	fake := func(p string, i io.Reader, o io.Writer, a ...string) error {
		if o != nil {
			o.Write([]byte("FAKE_OUTPUT\n"))
		}
		return nil
	}
	registry.RunCommandFunc = fake
	if err := imageV2.Copy(); err != nil {
		t.Error(err.Error())
	}
	if !imageV2.Copied {
		t.Error("Copy failed")
	}

	var imageV1 *Image = NewImage(&ImageOptions{
		Source:          "docker.io/example",
		Destination:     "private.io/library/example",
		Tag:             "v1.0.0",
		Arch:            "arm64",
		Variant:         "v8",
		OS:              "linux",
		Digest:          "sha256:" + u.Sha256Sum("ABC"),
		SourceMediaType: manifest.DockerV2Schema2MediaType,
		MID:             1,
	})
	if err := imageV1.Copy(); err != nil {
		t.Error(err.Error())
	}
	if !imageV1.Copied {
		t.Error("Copy failed")
	}

	// return random output, this will make source digest not equal to dest
	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		// sleep 10ms
		time.Sleep(time.Microsecond * 10)
		if o != nil {
			o.Write([]byte(time.Now().Format(time.StampNano) + "\n"))
		}
		return nil
	}
	registry.RunCommandFunc = fake

	var imageListV2 *Image = NewImage(&ImageOptions{
		Source:          "docker.io/example",
		Destination:     "private.io/library/example",
		Tag:             "v1.0.0",
		Arch:            "arm64",
		Variant:         "v8",
		OS:              "linux",
		Digest:          "sha256:" + u.Sha256Sum("ABC"),
		SourceMediaType: manifest.DockerV2ListMediaType,
		MID:             1,
	})
	if err := imageListV2.Copy(); err != nil {
		t.Error(err.Error())
	}
	if !imageListV2.Copied {
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
	if CopiedTag("1", "windows", "amd64", "", "1.0.1234") !=
		"1-windows-amd64-1.0.1234" {
		t.Error("CopiedTag failed")
	}
}

func Test_Load(t *testing.T) {
	img := Image{
		Source:      ".", // source is a directory
		Destination: "priv.io/library/nginx",
	}
	// fake skopeo copy function
	fake := func(a string, i io.Reader, o io.Writer, p ...string) error {
		return nil
	}
	registry.RunCommandFunc = fake
	if err := img.Load(); err != nil {
		t.Fatal(err)
	}
	if !img.Loaded {
		t.Error("load failed")
	}
	registry.RunCommandFunc = nil
}

func Test_Save(t *testing.T) {
	img := Image{
		Source:      "priv.io/library/nginx",
		Destination: ".", // dest is a directory
		Directory:   ".",
	}
	// fake skopeo copy function
	fake := func(a string, i io.Reader, o io.Writer, p ...string) error {
		return nil
	}
	registry.RunCommandFunc = fake
	if err := img.Save(); err != nil {
		t.Fatal(err)
	}
	if !img.Saved {
		t.Error("load failed")
	}
	registry.RunCommandFunc = nil
}
