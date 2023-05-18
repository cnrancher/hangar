package mirror

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/cnrancher/hangar/pkg/mirror/image"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/manifest"
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
	Digest    string
	Arch      string
	OS        string
	OsVersion string
	Variant   string
	Folder    string
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

func (s *SavedListTemplate) Has(mT *SavedMirrorTemplate) bool {
	if s == nil || mT == nil {
		return false
	}
	for _, v := range s.List {
		if reflect.DeepEqual(v, *mT) {
			return true
		}
	}
	return false
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
			Digest:    img.Digest,
			Arch:      img.Arch,
			Variant:   img.Variant,
			OS:        img.OS,
			OsVersion: img.OsVersion,
			Folder:    img.SavedFolder,
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
	if directory, err = utils.GetAbsPath(directory); err != nil {
		return nil, fmt.Errorf("LoadSavedTemplates: %w", err)
	}
	logrus.Debugf("LoadSavedTemplates from dir: %v", directory)

	savedList := SavedListTemplate{}
	f, err := os.Open(filepath.Join(directory, utils.SavedImageListFile))
	if err != nil {
		return nil, fmt.Errorf("LoadSavedTemplates: %w", err)
	}
	err = json.NewDecoder(f).Decode(&savedList)
	if err != nil {
		return nil, fmt.Errorf("LoadSavedTemplates: %w", err)
	}

	logrus.Debugf("savedList.Version: %v", savedList.Version)
	sVersion := savedList.Version
	sVersion, err = utils.EnsureSemverValid(sVersion)
	if err != nil {
		return nil, fmt.Errorf("LoadSavedTemplates: %w", err)
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
		if utils.GetProjectName(source) == "" && proj != "" {
			logrus.Warnf("%q does not have project name, set to %q",
				source, proj)
			source = utils.ReplaceProjectName(source, proj)
		}
		m := NewMirror(&MirrorOptions{
			Source:      mT.Source,
			Destination: utils.ConstructRegistry(source, destReg),
			Directory:   directory,
			Tag:         mT.Tag,
			ArchList:    mT.ArchList,
			Line:        fmt.Sprintf("%s:%s", mT.Source, mT.Tag),
			Mode:        MODE_LOAD,
			ID:          i + 1,
		})

		for _, iT := range mT.Images {
			var copiedTag string
			if iT.OsVersion == "" {
				copiedTag = image.CopiedTag(mT.Tag, iT.OS, iT.Arch, iT.Variant)
			} else {
				copiedTag = image.CopiedTag(
					mT.Tag, iT.OS, iT.Arch, iT.Variant, iT.OsVersion)
			}
			// Source is a directory
			srcImageDir := filepath.Join(directory, iT.Folder)
			// Destination is the dest registry
			repo := utils.ConstructRegistry(source, destReg)
			destImage := fmt.Sprintf("%s:%s", repo, copiedTag)
			img := image.NewImage(&image.ImageOptions{
				Source:      srcImageDir,
				Destination: destImage,
				// Directory is the decompressed folder path
				Directory:       directory,
				Tag:             mT.Tag,
				Arch:            iT.Arch,
				Variant:         iT.Variant,
				OS:              iT.OS,
				OsVersion:       iT.OsVersion,
				SavedFolder:     iT.Folder,
				SourceMediaType: manifest.DockerV2Schema2MediaType,
			})
			m.AppendImage(img)
		}
		mirrorList = append(mirrorList, m)
	}

	return mirrorList, nil
}
