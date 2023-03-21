package mirror

import (
	"embed"
	"io"
	"testing"

	"github.com/cnrancher/hangar/pkg/image"
	r "github.com/cnrancher/hangar/pkg/registry"
	u "github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/manifest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

const (
	TestS1V2FileName                = "test/s1v2.json"
	TestS2V2FileName                = "test/s2v2.json"
	TestS2V2ListFileName            = "test/s2v2-list.json"
	TestImageConfigFileName         = "test/s2v2-config.json"
	TestOCIIndexFileName            = "test/oci-index.json"
	TestOCIManifestFileName         = "test/oci-manifest.json"
	TestOCIMirroredManifestFileName = "test/oci-dst-manifest-list.json"
)

//go:embed test/*
var testFs embed.FS

func init() {
	logrus.SetOutput(io.Discard)
	u.WorkerNum = 2
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
	fake := func(p string, i io.Reader, o io.Writer, a ...string) error {
		// inspect func return S2V2 json manifest
		s2v2, err := testFs.ReadFile(TestS2V2FileName)
		if err != nil {
			return err
		}
		if o != nil {
			o.Write(s2v2)
		}
		return nil
	}
	r.RunCommandFunc = fake
	if err := m.initSourceDestinationManifest(); err != nil {
		t.Error("initSourceDestinationManifest failed:", err.Error())
	}
	// test initImageListByV2, read the configuration of the source image
	// to get the arch, os, calculate source manifest digest
	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		// inspect func return S2V2 json manifest
		s2v2, err := testFs.ReadFile(TestImageConfigFileName)
		if err != nil {
			return err
		}
		if o != nil {
			o.Write(s2v2)
		}
		return nil
	}
	r.RunCommandFunc = fake
	if err := m.initImageListByV2(); err != nil {
		t.Error("initImageListByV2 failed:", err.Error())
	}
	// reset the override command function
	r.RunCommandFunc = nil

	// test MIME type
	if m.sourceMIMEType != manifest.DockerV2Schema2MediaType {
		t.Errorf("sourceMIMEType failed: %v", m.sourceMIMEType)
	}

	// test manifest schemaVersion
	if m.sourceSchema2.SchemaVersion != 2 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v",
			m.sourceSchema2.SchemaVersion)
	}
	// test manifest mediaType
	if m.sourceSchema2.MediaType != manifest.DockerV2Schema2MediaType {
		t.Errorf("sourceManifestMediaType failed, mediaType: %v",
			m.sourceSchema2.MediaType)
	}

	// fake skopeo copy function
	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		if o != nil {
			o.Write([]byte("FAKE_OUTPUT\n"))
		}
		return nil
	}
	r.RunCommandFunc = fake
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
	srcSpec := m.SourceManifestSpec()
	if len(srcSpec) != 1 {
		t.Error("SourceManifestSpec failed")
		return
	}
	// output should be the sha256sum of the source manifest
	srcManifest, _ := testFs.ReadFile(TestS2V2FileName)
	sourceSum := "sha256:" + u.Sha256Sum(string(srcManifest[:]))
	if srcSpec[0].Digest != sourceSum {
		t.Errorf("SourceDigests should be %q, but got %q",
			sourceSum, srcSpec[0].Digest)
	}

	// destination mediaType is manifest.v2, do not have digest list
	// should get empty slice
	if list := m.DestinationManifestSpec(); len(list) != 0 {
		t.Error("DestinationManifestSpec failed")
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

	// if dest manifest does not exists, dest MIME type is empty str
	m.destMIMEType = ""
	if list := m.DestinationManifestSpec(); len(list) != 0 {
		t.Error("DestinationManifestSpec failed")
	}
	if m.compareSourceDestManifest() {
		// dest manifest is nil
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	// Reset the override run command func
	r.RunCommandFunc = nil
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
	fake := func(path string, i io.Reader, o io.Writer, args ...string) error {
		// inspect func return S2V2 json manifest.list
		s2v2, err := testFs.ReadFile(TestS2V2ListFileName)
		if err != nil {
			return err
		}
		if o != nil {
			o.Write(s2v2)
		}
		return nil
	}
	r.RunCommandFunc = fake
	if err := m.initSourceDestinationManifest(); err != nil {
		t.Error("initSourceDestinationManifest failed:", err.Error())
	}
	r.RunCommandFunc = nil

	// test MIME Type
	if m.sourceMIMEType != manifest.DockerV2ListMediaType {
		t.Errorf("sourceMIMEType failed: %v", m.sourceMIMEType)
	}

	if m.sourceSchema2List.SchemaVersion != 2 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v",
			m.sourceSchema2.SchemaVersion)
	}
	if m.sourceSchema2List.MediaType != manifest.DockerV2ListMediaType {
		t.Errorf("sourceManifestMediaType failed, mediaType: %v",
			m.sourceSchema2List.MediaType)
	}

	// generate images from source manifest list
	if err := m.initSourceImageListByListV2(); err != nil {
		t.Error("initSourceImageListByListV2 failed:", err.Error())
	}

	// simulate copy operation
	// fake skopeo copy function
	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		if o != nil {
			o.Write([]byte("FAKE_OUTPUT"))
		}
		return nil
	}
	r.RunCommandFunc = fake
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
	srcSpec := m.SourceManifestSpec()
	if len(srcSpec) == 0 {
		t.Error("SourceManifestSpec failed")
		return
	}

	// destination mediaType is manifest.list.v2, should have multi-digests
	dstSpec := m.DestinationManifestSpec()
	if len(dstSpec) == 0 {
		t.Error("DestinationManifestSpec failed")
	}
	if len(srcSpec) != len(dstSpec) {
		t.Errorf("len(srcSpec): %d, len(dstSpec): %d",
			len(srcSpec), len(dstSpec))
		t.Errorf("srcSpec: %+v", srcSpec)
		t.Errorf("dstSpec: %+v", dstSpec)
		t.Error("the length of srcDigests and dstDigests should be same")
	}

	// source digests should equal to the dest digests
	if !m.compareSourceDestManifest() {
		// dest schemaVersion is 2, mediatype is manifest.list.v2
		// source image is copied to dest image
		// should return true
		t.Error("compareSourceDestManifest failed")
	}

	m.destMIMEType = ""
	if m.compareSourceDestManifest() {
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	// Reset the override run command func
	r.RunCommandFunc = nil
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
	fake := func(p string, i io.Reader, o io.Writer, a ...string) error {
		// inspect func return S1V2 json manifest
		s1v2, err := testFs.ReadFile(TestS1V2FileName)
		if err != nil {
			return err
		}
		if o != nil {
			o.Write(s1v2)
		}
		return nil
	}
	r.RunCommandFunc = fake
	if err := m.initSourceDestinationManifest(); err != nil {
		t.Error("initSourceDestinationManifest:", err.Error())
	}

	if m.sourceMIMEType != manifest.DockerV2Schema1MediaType &&
		m.sourceMIMEType != manifest.DockerV2Schema1SignedMediaType {
		t.Errorf("sourceMIMEType failed: %v", m.sourceMIMEType)
	}

	if m.sourceSchema1.SchemaVersion != 1 {
		t.Errorf("SchemaVersion failed, version: %v",
			m.sourceSchema1.SchemaVersion)
	}

	if m.sourceMIMEType != manifest.DockerV2Schema1MediaType &&
		m.sourceMIMEType != manifest.DockerV2Schema1SignedMediaType {
		t.Errorf("sourceMIMEType failed, sourceMIMEType %v", m.sourceMIMEType)
	}

	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		// inspect func return S1V2 json manifest
		s1v2, err := testFs.ReadFile(TestS1V2FileName)
		if err != nil {
			return err
		}
		if o != nil {
			o.Write(s1v2)
		}
		return nil
	}
	r.RunCommandFunc = fake
	// Generate imager from source manifest
	if err := m.initImageListByV1(); err != nil {
		t.Error("initImageListByV2 failed:", err.Error())
	}
	r.RunCommandFunc = nil

	if m.ImageNum() != 1 {
		t.Error("initImageListByV1 should only generate 1 image")
		return
	}

	// simulate copy operation
	// fake skopeo copy and skopeo inspect function
	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		if o != nil {
			o.Write([]byte("FAKE_OUTPUT"))
		}
		return nil
	}
	r.RunCommandFunc = fake
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

	srcSpec := m.SourceManifestSpec()
	if len(srcSpec) != 1 {
		t.Error("SourceManifestSpec failed")
		return
	}
	if srcSpec[0].Digest != "sha256:"+u.Sha256Sum("FAKE_OUTPUT") {
		t.Error("SourceDigests should be the sha256sum of 'FAKE_OUTPUT'")
	}
	// dest schemaVersion is 1, should return empty slice
	dstSpec := m.DestinationManifestSpec()
	if len(dstSpec) != 0 {
		t.Error("DestinationManifestSpec failed")
	}

	if m.compareSourceDestManifest() {
		// dest schemaVersion is 1
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	m.destMIMEType = ""
	if m.compareSourceDestManifest() {
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	// docker buildx
	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		return nil
	}
	r.RunCommandFunc = fake
	// updateDestManifest
	if err := m.updateDestManifest(); err != nil {
		t.Error("updateDestManifest:", err.Error())
	}
	r.RunCommandFunc = nil
}

// Test_OCI_Index simulates the mirror operations when
// source image mediaType is OCI "application/vnd.oci.image.index.v1+json"
func Test_OCI_Index(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		ArchList:    []string{"amd64", "arm64"},
		Mode:        MODE_MIRROR,
	})

	// make both source manifest and dest manifest are schemaVersion V2,
	// mediaType manifest.list.v2
	fake := func(path string, i io.Reader, o io.Writer, args ...string) error {
		// inspect func return S2V2 json manifest.list
		s2v2, err := testFs.ReadFile(TestOCIIndexFileName)
		if err != nil {
			return err
		}
		if o != nil {
			o.Write(s2v2)
		}
		return nil
	}
	r.RunCommandFunc = fake
	if err := m.initSourceDestinationManifest(); err != nil {
		t.Error("initSourceDestinationManifest failed:", err.Error())
	}
	r.RunCommandFunc = nil

	// test MIME Type
	if m.sourceMIMEType != imgspecv1.MediaTypeImageIndex {
		t.Errorf("sourceMIMEType failed: %v", m.sourceMIMEType)
	}

	if m.sourceOCIIndex.SchemaVersion != 2 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v",
			m.sourceOCIIndex.SchemaVersion)
	}
	if m.sourceOCIIndex.MediaType != imgspecv1.MediaTypeImageIndex {
		t.Errorf("sourceManifestMediaType failed, mediaType: %v",
			m.sourceOCIIndex.MediaType)
	}

	// generate images from source manifest list
	if err := m.initSourceImageListByOCIIndexV1(); err != nil {
		t.Error("initSourceImageListByOCIIndexV1 failed:", err.Error())
	}

	// simulate copy operation
	// fake skopeo copy function
	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		if o != nil {
			o.Write([]byte("FAKE_OUTPUT"))
		}
		return nil
	}
	r.RunCommandFunc = fake
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

	// source manifest mediaType is oci.image.index.v1, should have multi-digests
	srcSpec := m.SourceManifestSpec()
	if len(srcSpec) == 0 {
		t.Error("SourceManifestSpec failed")
		return
	}

	out, err := testFs.ReadFile(TestOCIMirroredManifestFileName)
	if err != nil {
		t.Fatal(err)
	}
	m.destMIMEType = manifest.GuessMIMEType(out)
	switch m.destMIMEType {
	case manifest.DockerV2ListMediaType: // schemaVersion 2 manifest.list.v2
		m.destSchema2List, err = manifest.Schema2ListFromManifest([]byte(out))
		if err != nil {
			t.Fatal(err.Error())
		}
	default:
		t.Fatal("dest MIME type:", m.destMIMEType)
		// ignore other MIME type
	}
	m.destManifestStr = string(out)

	// destination mediaType is Docker manifest.list.v2, should have multi-digests
	dstSpec := m.DestinationManifestSpec()
	if len(dstSpec) == 0 {
		t.Error("DestinationManifestSpec failed")
	}
	if len(srcSpec) != len(dstSpec) {
		t.Errorf("len(srcSpec): %d, len(dstSpec): %d",
			len(srcSpec), len(dstSpec))
		t.Errorf("srcSpec: %+v", srcSpec)
		t.Errorf("dstSpec: %+v", dstSpec)
		t.Error("the length of srcDigests and dstDigests should be same")
	}

	// source digests should equal to the dest digests
	if !m.compareSourceDestManifest() {
		// dest schemaVersion is 2, mediatype is manifest.list.v2
		// source image is copied to dest image
		// should return true
		t.Error("compareSourceDestManifest failed")
		srcSpecs := m.SourceManifestSpec()
		dstSpecs := m.DestinationManifestSpec()
		t.Errorf("srcSpecs: %++v", srcSpecs)
		t.Errorf("dstSpecs: %++v", dstSpecs)
	}

	m.destMIMEType = ""
	if m.compareSourceDestManifest() {
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	// Reset the override run command func
	r.RunCommandFunc = nil
}

// Test_OCI_Manifest simulates the mirror operations when
// source image mediaType is "application/vnd.oci.image.manifest.v1+json".
func Test_OCI_Manifest(t *testing.T) {
	m := NewMirror(&MirrorOptions{
		Source:      "registry.io/example",
		Destination: "private.io/example",
		Tag:         "v1.0.0",
		ArchList:    []string{"amd64", "arm64"},
		Mode:        MODE_MIRROR,
	})

	// test initSourceDestinationManifest, make both source manifest and dest
	// manifest are schemaVersion V2, mediaType manifest.v2
	fake := func(p string, i io.Reader, o io.Writer, a ...string) error {
		// inspect func return S2V2 json manifest
		s2v2, err := testFs.ReadFile(TestOCIManifestFileName)
		if err != nil {
			return err
		}
		if o != nil {
			o.Write(s2v2)
		}
		return nil
	}
	r.RunCommandFunc = fake
	if err := m.initSourceDestinationManifest(); err != nil {
		t.Error("initSourceDestinationManifest failed:", err.Error())
	}
	// test initImageListByV2, read the configuration of the source image
	// to get the arch, os, calculate source manifest digest
	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		// inspect func return S2V2 json manifest
		s2v2, err := testFs.ReadFile(TestImageConfigFileName)
		if err != nil {
			return err
		}
		if o != nil {
			o.Write(s2v2)
		}
		return nil
	}
	r.RunCommandFunc = fake
	if err := m.initImageListByOCIManifestV1(); err != nil {
		t.Error("initImageListByOCIManifestV1 failed:", err.Error())
	}
	// reset the override command function
	r.RunCommandFunc = nil

	// test MIME type
	if m.sourceMIMEType != imgspecv1.MediaTypeImageManifest {
		t.Errorf("sourceMIMEType failed: %v", m.sourceMIMEType)
	}

	// test manifest schemaVersion
	if m.sourceOCIManifest.SchemaVersion != 2 {
		t.Errorf("sourceManifestSchemaVersion failed, version: %v",
			m.sourceOCIManifest.SchemaVersion)
	}
	// test manifest mediaType
	if m.sourceOCIManifest.MediaType != imgspecv1.MediaTypeImageManifest {
		t.Errorf("sourceManifestMediaType failed, mediaType: %v",
			m.sourceOCIManifest.MediaType)
	}

	// fake skopeo copy function
	fake = func(p string, i io.Reader, o io.Writer, a ...string) error {
		if o != nil {
			o.Write([]byte("FAKE_OUTPUT\n"))
		}
		return nil
	}
	r.RunCommandFunc = fake
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
	srcSpec := m.SourceManifestSpec()
	if len(srcSpec) != 1 {
		t.Error("SourceManifestSpec failed")
		return
	}
	// output should be the sha256sum of the source manifest
	srcManifest, _ := testFs.ReadFile(TestOCIManifestFileName)
	sourceSum := "sha256:" + u.Sha256Sum(string(srcManifest[:]))
	if srcSpec[0].Digest != sourceSum {
		t.Errorf("SourceDigests should be %q, but got %q",
			sourceSum, srcSpec[0].Digest)
	}

	// destination mediaType is manifest.v2, do not have digest list
	// should get empty slice
	if list := m.DestinationManifestSpec(); len(list) != 0 {
		t.Error("DestinationManifestSpec failed")
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

	// if dest manifest does not exists, dest MIME type is empty str
	m.destMIMEType = ""
	if list := m.DestinationManifestSpec(); len(list) != 0 {
		t.Error("DestinationManifestSpec failed")
	}
	if m.compareSourceDestManifest() {
		// dest manifest is nil
		// should return false
		t.Error("compareSourceDestManifest failed")
	}

	// Reset the override run command func
	r.RunCommandFunc = nil
}
