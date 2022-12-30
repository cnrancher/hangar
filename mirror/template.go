package mirror

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cnrancher.io/image-tools/image"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	"golang.org/x/mod/semver"
)

const (
	SavedTemplateVersion = "v1.1.0"
)

type SavedListTemplate struct {
	List      []SavedMirrorTemplate
	Version   string
	SavedTime string
}

type SavedMirrorTemplate struct {
	Source   string
	Tag      string
	ArchList []string
	Images   []SavedImagesTemplate
}

type SavedImagesTemplate struct {
	Arch    string
	OS      string
	Variant string
	Folder  string
}

type DockerBuildxManifest struct {
	Digest   string               `json:"digest"`
	Platform DockerBuildxPlatform `json:"platform"`
}

type DockerBuildxPlatform struct {
	Architecture string `json:"architecture,omitempty"`
	OS           string `json:"os,omitempty"`
	Variant      string `json:"variant,omitempty"`
	OsVersion    string `json:"os.version,omitempty"`
}

func NewSavedListTemplate() *SavedListTemplate {
	return &SavedListTemplate{
		List:      make([]SavedMirrorTemplate, 0),
		Version:   SavedTemplateVersion,
		SavedTime: time.Now().Format(time.RFC3339),
	}
}

func (s *SavedListTemplate) Append(mT *SavedMirrorTemplate) {
	if mT == nil {
		return
	}
	s.List = append(s.List, *mT)
}

func (m *Mirror) GetSavedImageTemplate() *SavedMirrorTemplate {
	if m.Mode != MODE_SAVE {
		return nil
	}

	mT := SavedMirrorTemplate{
		Source:   m.Source,
		Tag:      m.Tag,
		ArchList: make([]string, 0),
		Images:   make([]SavedImagesTemplate, 0),
	}
	for _, img := range m.images {
		if mT.ArchList == nil ||
			!slices.Contains(mT.ArchList, img.Arch) {
			mT.ArchList = append(mT.ArchList, img.Arch)
		}
		iT := SavedImagesTemplate{
			Arch:    img.Arch,
			OS:      img.OS,
			Variant: img.Variant,
			Folder:  img.SavedFolder,
		}
		mT.Images = append(mT.Images, iT)
	}
	if len(mT.ArchList) == 0 {
		return nil
	}

	return &mT
}

// LoadSavedTemplates loads the saved json templates to *Mirror slice
func LoadSavedTemplates(directory, destReg, proj string) ([]*Mirror, error) {
	var err error
	if directory, err = u.GetAbsPath(directory); err != nil {
		return nil, fmt.Errorf("LoadSavedMirrorTemplate: %w", err)
	}
	logrus.Debugf("LoadSavedTemplates from dir: %v", directory)

	savedList := SavedListTemplate{}
	f, err := os.Open(filepath.Join(directory, u.SavedImageListFile))
	if err != nil {
		return nil, fmt.Errorf("LoadSavedMirrorTemplate: %w", err)
	}
	err = json.NewDecoder(f).Decode(&savedList)
	if err != nil {
		return nil, fmt.Errorf("LoadSavedMirrorTemplate: %w", err)
	}

	logrus.Debugf("savedList.Version: %v", savedList.Version)
	sVersion := savedList.Version
	if !strings.HasPrefix(sVersion, "v") {
		sVersion = "v" + sVersion
	}
	if semver.Compare(sVersion, SavedTemplateVersion) != 0 {
		logrus.Warnf("Template version in saved tarball is %q", sVersion)
		logrus.Warnf("The template version supported of this tool is %q",
			SavedTemplateVersion)
		return nil, fmt.Errorf(
			"this tool does not support template version %q",
			sVersion)
	}

	var mirrorList []*Mirror
	for i, mT := range savedList.List {
		source := mT.Source
		if u.GetProjectName(source) == "" && proj != "" {
			logrus.Warnf("%q does not have project name, set to %q",
				source, proj)
			source = u.ReplaceProjectName(source, proj)
		}
		m := NewMirror(&MirrorOptions{
			Source:      mT.Source,
			Destination: u.ConstructRegistry(source, destReg),
			Directory:   directory,
			Tag:         mT.Tag,
			ArchList:    mT.ArchList,
			Mode:        MODE_LOAD,
			ID:          i + 1,
		})

		for _, iT := range mT.Images {
			copiedTag := image.CopiedTag(mT.Tag, iT.OS, iT.Arch, iT.Variant)
			// Source is a directory
			srcImageDir := filepath.Join(directory, iT.Folder)
			// Destination is the dest registry
			repo := u.ConstructRegistry(source, destReg)
			destImage := fmt.Sprintf("%s:%s", repo, copiedTag)
			img := image.NewImage(&image.ImageOptions{
				Source:      srcImageDir,
				Destination: destImage,
				// Directory is the decompressed folder path
				Directory:   directory,
				Tag:         mT.Tag,
				Arch:        iT.Arch,
				Variant:     iT.Variant,
				OS:          iT.OS,
				SavedFolder: iT.Folder,

				// saved image manifest is already converted to v2s2
				SourceSchemaVersion: 2,
				SourceMediaType:     u.MediaTypeManifestV2,
			})
			m.AppendImage(img)
		}
		mirrorList = append(mirrorList, m)
	}

	return mirrorList, nil
}

func CompareBuildxManifests(src, dst []DockerBuildxManifest) bool {
	if src == nil || dst == nil {
		return false
	}
	if len(src) != len(dst) {
		return false
	}
	for i := range src {
		if !CompareBuildxManifest(&src[i], &dst[i]) {
			return false
		}
	}
	return true
}

func CompareBuildxManifest(src, dst *DockerBuildxManifest) bool {
	if src == nil || dst == nil {
		return false
	}
	if src.Digest != dst.Digest {
		return false
	}
	if src.Platform.Architecture != dst.Platform.Architecture {
		return false
	}
	if src.Platform.OS != dst.Platform.OS {
		return false
	}
	if src.Platform.Variant != dst.Platform.Variant {
		return false
	}
	if src.Platform.OsVersion != dst.Platform.OsVersion {
		return false
	}
	return true
}
