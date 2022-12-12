package mirror

import (
	"embed"
	"io"
	"testing"

	"cnrancher.io/image-tools/image"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

const (
	TestS1V2FileName     = "test/s1v2.json"
	TestS1V2RepoFileName = "test/s1v2-repo.json"
	TestS2V2FileName     = "test/s2v2.json"
	TestS2V2ListFileName = "test/s2v2-list.json"
	TestS2V2OciFileName  = "test/s2v2-oci.json"
)

//go:embed test/*
var testFs embed.FS

func init() {
	logrus.SetOutput(io.Discard)
}

// StartMirror method should test manually,
// this method can not be implemented in unit test

func Test_NewMirror(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		Directory:   ".saved-cache",
		ArchList:    []string{"amd64", "arm64"},
		Mode:        MODE_MIRROR,
	})

	if m.Source != "registry.io/example" {
		t.Error("Source failed")
	}
	if m.Destination != "private.io/example" {
		t.Error("Destination failed")
	}
	if m.Tag != "v1.0.0" {
		t.Error("Tag failed")
	}
	if m.Directory != ".saved-cache" {
		t.Error("Directory failed")
	}
	if !m.HasArch("amd64") || m.HasArch("s390x") {
		t.Error("HasArch failed")
	}
	if m.Mode != MODE_MIRROR {
		t.Error("Mode failed")
	}
	img := image.NewImage(&image.ImageOptions{})
	m.AppendImage(img)
	if m.ImageNum() != 1 {
		t.Error("AppendImage failed")
	}
	if m.Copied() != 0 {
		t.Error("Copied failed")
	}
	if m.ImageNum()-m.Copied() != 1 {
		t.Error("CopyFailed failed")
	}
}

// Test_S2V2 simulates the mirror operations when
// source image mediaType is manifest.v2.
func Test_S2V2(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		ArchList:    []string{"amd64", "arm64"},
		Mode:        MODE_MIRROR,
	})

	// test initSourceDestinationManifest, make both source manifest and dest
	// manifest are schemaVersion V2, mediaType manifest.v2
	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		// inspect func return S2V2 json manifest
		s2v2, err := testFs.ReadFile(TestS2V2FileName)
		return string(s2v2[:]), err
	}
	if err := m.initSourceDestinationManifest(); err != nil {
		t.Error("initSourceDestinationManifest failed:", err.Error())
	}
	// test initImageListByV2, read the configuration of the source image
	// to get the arch, os, calculate source manifest digest
	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		// inspect func return S2V2 json manifest
		s2v2, err := testFs.ReadFile(TestS2V2OciFileName)
		return string(s2v2[:]), err
	}
	if err := m.initImageListByV2(); err != nil {
		t.Error("initImageListByV2 failed:", err.Error())
	}
	// reset the override command function
	registry.RunCommandFunc = nil

	// test manifest schemaVersion
	if v, err := m.sourceManifestSchemaVersion(); err != nil || v != 2 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v", v)
		t.Error(err.Error())
	}
	// test manifest mediaType
	if m, err := m.sourceManifestMediaType(); err != nil ||
		m != u.MediaTypeManifestV2 {
		t.Errorf("sourceManifestMediaType failed, mediaType: %v", m)
		t.Error(err.Error())
	}

	// fake skopeo copy function
	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		return "FAKE_OUTPUT\n", nil
	}
	for _, img := range m.images {
		if err := img.Copy(); err != nil {
			t.Error("img.Copy failed:", err.Error())
			return
		}
		if !img.Copied {
			t.Error("img.Copied failed")
			return
		}
	}
	if m.Copied() != 1 {
		t.Error("m.Copied should be 1")
	}
	// now the image status should be set to copied
	// compare the source digests and dest digests

	// source manifest mediaType is manifest.v2, should have one digest
	list := m.SourceDigests()
	if len(list) != 1 {
		t.Error("SourceDigests failed")
		return
	}
	// output should be the sha256sum of the source manifest
	srcManifest, _ := testFs.ReadFile(TestS2V2FileName)
	sourceSum := "sha256:" + u.Sha256Sum(string(srcManifest[:]))
	if list[0] != sourceSum {
		t.Errorf("SourceDigests should be %s, but got %s", sourceSum, list[0])
	}

	// destination mediaType is manifest.v2, do not have digest list
	// should get empty slice
	if list := m.DestinationDigests(); len(list) != 0 {
		t.Error("DestinationDigests failed")
	}

	// dest schemaVersion is 2, mediatype is manifest.v2
	// should return false, (then create new manifest list for dest)
	if m.compareSourceDestManifest() {
		t.Error("compareSourceDestManifest failed")
	}

	// simulate the docker buildx operation
	if err := m.updateDestManifest(); err != nil {
		t.Error("updateDestManifest failed:", err.Error())
	}
	// now the mirror operation of s2v2 is finished

	// if dest image does not exists, destManifest is nil
	m.destManifest = nil
	if list := m.DestinationDigests(); len(list) != 0 {
		t.Error("DestinationDigests failed")
	}
	if m.compareSourceDestManifest() {
		// dest manifest is nil
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	// Reset the override run command func
	registry.RunCommandFunc = nil
}

// Test_S2V2List simulates the mirror operations when
// source image mediaType is manifest.list.v2.
func Test_S2V2List(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		ArchList:    []string{"amd64", "arm64"},
		Mode:        MODE_MIRROR,
	})

	// make both source manifest and dest manifest are schemaVersion V2,
	// mediaType manifest.list.v2
	testInspectFunc := func(path string, args ...string) (string, error) {
		// inspect func return S2V2 json manifest.list
		s2v2, err := testFs.ReadFile(TestS2V2ListFileName)
		return string(s2v2[:]), err
	}
	registry.RunCommandFunc = testInspectFunc
	if err := m.initSourceDestinationManifest(); err != nil {
		t.Error("initSourceDestinationManifest failed:", err.Error())
	}
	registry.RunCommandFunc = nil

	if v, err := m.sourceManifestSchemaVersion(); err != nil || v != 2 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v", v)
		t.Error(err.Error())
	}
	if m, err := m.sourceManifestMediaType(); err != nil ||
		m != u.MediaTypeManifestListV2 {
		t.Errorf("sourceManifestMediaType failed, mediaType: %v", m)
		t.Error(err.Error())
	}

	// generate images from source manifest list
	if err := m.initImageListByListV2(); err != nil {
		t.Error("initImageListByV2 failed:", err.Error())
	}

	// simulate copy operation
	// fake skopeo copy function
	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		return "FAKE_OUTPUT", nil
	}
	for _, img := range m.images {
		if err := img.Copy(); err != nil {
			t.Error("img.Copy failed:", err.Error())
			return
		}
		if !img.Copied {
			t.Error("img.Copied failed")
			return
		}
	}
	if m.Copied() != 2 {
		t.Error("m.Copied should be 2")
	}
	// now the image status should be set to copied
	// compare the source digests and dest digests

	// source manifest mediaType is manifest.list.v2, should have multi-digests
	srcDigests := m.SourceDigests()
	if len(srcDigests) == 0 {
		t.Error("SourceDigests failed")
		return
	}

	// destination mediaType is manifest.list.v2, should have multi-digests
	dstDigests := m.DestinationDigests()
	if len(dstDigests) == 0 {
		t.Error("DestinationDigests failed")
	}
	if len(srcDigests) != len(dstDigests) {
		t.Error("the length of srcDigests and dstDigests should be same")
	}

	// source digests should equal to the dest digests
	if !m.compareSourceDestManifest() {
		// dest schemaVersion is 2, mediatype is manifest.list.v2
		// source image is copied to dest image
		// should return true
		t.Error("compareSourceDestManifest failed")
	}

	m.destManifest = nil
	if m.compareSourceDestManifest() {
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	// Reset the override run command func
	registry.RunCommandFunc = nil
}

// Test_S1V2 simulates the mirror operations when
// source image deprecated schemaVersion V1.
func Test_S1V2(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		ArchList:    []string{"amd64", "arm64"},
		Mode:        MODE_MIRROR,
	})

	// set source & dest manifest to same s1v2
	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		// inspect func return S1V2 json manifest
		s1v2, err := testFs.ReadFile(TestS1V2FileName)
		return string(s1v2[:]), err
	}
	if err := m.initSourceDestinationManifest(); err != nil {
		t.Error("initSourceDestinationManifest:", err.Error())
	}

	if v, err := m.sourceManifestSchemaVersion(); err != nil || v != 1 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v", v)
		t.Error(err.Error())
	}

	if m, err := m.sourceManifestMediaType(); err == nil || m != "" {
		t.Errorf("sourceManifestMediaType failed, mediaType should be empty")
	}

	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		// inspect func return S1V2 json manifest
		s1v2, err := testFs.ReadFile(TestS1V2RepoFileName)
		return string(s1v2[:]), err
	}
	// Generate imager from source manifest
	if err := m.initImageListByV1(); err != nil {
		t.Error("initImageListByV2 failed:", err.Error())
	}
	registry.RunCommandFunc = nil

	if m.ImageNum() != 1 {
		t.Error("initImageListByV1 should only generate 1 image")
		return
	}

	// simulate copy operation
	// fake skopeo copy and skopeo inspect function
	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		return "FAKE_OUTPUT", nil
	}
	for _, img := range m.images {
		if err := img.Copy(); err != nil {
			t.Error("img.Copy failed:", err.Error())
			return
		}
		if !img.Copied {
			t.Error("img.Copied failed")
			return
		}
	}
	if m.Copied() != 1 {
		t.Error("m.Copied should be 1")
	}
	// now the image status should be set to copied
	// compare the source digests and dest digests

	srcDigests := m.SourceDigests()
	if len(srcDigests) != 1 {
		t.Error("SourceDigests failed")
		return
	}
	if srcDigests[0] != "sha256:"+u.Sha256Sum("FAKE_OUTPUT") {
		t.Error("SourceDigests should be the sha256sum of 'FAKE_OUTPUT'")
	}
	// dest schemaVersion is 1, should return empty slice
	dstDigests := m.DestinationDigests()
	if len(dstDigests) != 0 {
		t.Error("dstDigests failed")
	}

	if m.compareSourceDestManifest() {
		// dest schemaVersion is 1
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	m.destManifest = nil
	if m.compareSourceDestManifest() {
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	// docker buildx
	registry.RunCommandFunc = func(p string, a ...string) (string, error) {
		return "", nil
	}
	// updateDestManifest
	if err := m.updateDestManifest(); err != nil {
		t.Error("updateDestManifest:", err.Error())
	}
	registry.RunCommandFunc = nil
}
