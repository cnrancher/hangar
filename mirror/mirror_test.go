package mirror

import (
	"bytes"
	"embed"
	"encoding/json"
	"testing"

	"cnrancher.io/image-tools/image"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

const (
	TestS1V2FileName     = "test/s1v2.json"
	TestS2V2FileName     = "test/s2v2.json"
	TestS2V2ListFileName = "test/s2v2-list.json"
	TestS2V2OciFileName  = "test/s2v2-oci.json"
)

//go:embed test/*
var testFs embed.FS

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestMirrorerInterface(t *testing.T) {
	mirror := NewMirror(&MirrorOptions{})
	var mirrorer Mirrorer = mirror
	_ = mirrorer
}

func TestConstructureRegistry(t *testing.T) {
	s := ConstructRegistry("nginx", "")
	if s != "docker.io/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = ConstructRegistry("docker.io/nginx", "")
	if s != "docker.io/nginx" {
		t.Error("value should be 'docker.io/nginx'")
	}

	s = ConstructRegistry("localhost/nginx", "")
	if s != "localhost/nginx" {
		t.Error("value should be 'localhost/nginx'")
	}

	s = ConstructRegistry("custom.io/nginx", "")
	if s != "custom.io/nginx" {
		t.Error("value should be 'custom.io/nginx'")
	}

	dstReg := "private.io"

	s = ConstructRegistry("nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}

	s = ConstructRegistry("docker.io/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}

	s = ConstructRegistry("localhost/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}

	s = ConstructRegistry("custom.io/nginx", dstReg)
	if s != dstReg+"/nginx" {
		t.Errorf("value should be '%s'", dstReg+"/nginx")
	}
}

func TestNewMirror(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		ArchList:    []string{"amd64", "arm64"},
	})
	var mirrorer Mirrorer = m

	if mirrorer.Source() != "registry.io/example" {
		t.Error("Source failed")
	}
	if mirrorer.Destination() != "private.io/example" {
		t.Error("Destination failed")
	}
	if mirrorer.Tag() != "v1.0.0" {
		t.Error("Tag failed")
	}
	if !mirrorer.HasArch("amd64") || mirrorer.HasArch("s390x") {
		t.Error("HasArch failed")
	}
	img := image.NewImage(&image.ImageOptions{})
	mirrorer.AppendImage(img)
	if mirrorer.ImageNum() != 1 {
		t.Error("AppendImage failed")
	}
	if mirrorer.Copied() != 0 {
		t.Error("Copied failed")
	}
	if mirrorer.Failed() != 1 {
		t.Error("Failed failed")
	}
}

func TestS2V2(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		ArchList:    []string{"amd64", "arm64"},
	})
	s2v2, err := testFs.ReadFile(TestS2V2FileName)
	if err != nil {
		t.Error("testFs.ReadFile failed")
		return
	}
	err = json.NewDecoder(bytes.NewReader(s2v2)).Decode(&m.sourceManifest)
	err = json.NewDecoder(bytes.NewReader(s2v2)).Decode(&m.destManifest)
	if err != nil {
		t.Error("Decode failed")
		return
	}
	var mirrorer Mirrorer = m
	_ = mirrorer

	if v, err := m.sourceManifestSchemaVersion(); err != nil || v != 2 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v", v)
		t.Error(err.Error())
	}
	if m, err := m.sourceManifestMediaType(); err != nil ||
		m != u.MediaTypeManifestV2 {
		t.Errorf("sourceManifestMediaType failed, mediaType: %v", m)
		t.Error(err.Error())
	}

	// TODO: test initImageListByV2
	// if err := m.initImageListByV2(); err != nil {
	// 	t.Error("initImageListByV2 failed:", err.Error())
	// }

	if m.compareSourceDestManifest() {
		// dest schemaVersion is 2, mediatype is manifest.v2
		// should return false
		t.Error("compareSourceDestManifest failed")
	}
	m.destManifest = nil
	if m.compareSourceDestManifest() {
		// dest manifest is nil
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	sourceDigests := m.SourceDigests()
	if sourceDigests == nil || len(sourceDigests) != 0 {
		t.Error("SourceDigests failed")
	}
}

func TestS2V2List(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		ArchList:    []string{"amd64", "arm64"},
	})
	s2v2List, err := testFs.ReadFile(TestS2V2ListFileName)
	if err != nil {
		t.Error("testFs.ReadFile failed")
		return
	}
	err = json.NewDecoder(bytes.NewReader(s2v2List)).Decode(&m.sourceManifest)
	err = json.NewDecoder(bytes.NewReader(s2v2List)).Decode(&m.destManifest)
	if err != nil {
		t.Error("Decode failed")
		return
	}
	var mirrorer Mirrorer = m
	_ = mirrorer

	if v, err := m.sourceManifestSchemaVersion(); err != nil || v != 2 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v", v)
		t.Error(err.Error())
	}
	if m, err := m.sourceManifestMediaType(); err != nil ||
		m != u.MediaTypeManifestListV2 {
		t.Errorf("sourceManifestMediaType failed, mediaType: %v", m)
		t.Error(err.Error())
	}

	if err := m.initImageListByListV2(); err != nil {
		t.Error("initImageListByV2 failed:", err.Error())
	}

	if m.compareSourceDestManifest() {
		// dest schemaVersion is 2, mediatype is manifest.list.v2
		// source image is not copied to dest image
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	m.destManifest = nil
	if m.compareSourceDestManifest() {
		// should return false
		t.Error("compareSourceDestManifest failed")
	}
}

func TestS1V2(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		ArchList:    []string{"amd64", "arm64"},
	})
	s1v2, err := testFs.ReadFile(TestS1V2FileName)
	if err != nil {
		t.Error("testFs.ReadFile failed")
		return
	}
	err = json.NewDecoder(bytes.NewReader(s1v2)).Decode(&m.sourceManifest)
	err = json.NewDecoder(bytes.NewReader(s1v2)).Decode(&m.destManifest)
	if err != nil {
		t.Error("Decode failed")
		return
	}

	if v, err := m.sourceManifestSchemaVersion(); err != nil || v != 1 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v", v)
		t.Error(err.Error())
	}

	if m, err := m.sourceManifestMediaType(); err == nil || m != "" {
		t.Errorf("sourceManifestMediaType failed, mediaType should be empty")
	}

	// if err := m.initImageListByV1(); err != nil {
	// 	t.Error("initImageListByV2 failed:", err.Error())
	// }

	if m.compareSourceDestManifest() {
		// dest schemaVersion is 2, mediatype is manifest.list.v2
		// source image is not copied to dest image
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	m.destManifest = nil
	if m.compareSourceDestManifest() {
		// should return false
		t.Error("compareSourceDestManifest failed")
	}
}

func TestSourceDigests(t *testing.T) {
}

func TestDestinationDigests(t *testing.T) {

}
