package mirror

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cnrancher.io/image-tools/image"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

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
	Digest  string
}

func (m *Mirror) GetSavedImageTemplate() *SavedMirrorTemplate {
	if m.mode != MODE_SAVE {
		return nil
	}

	mT := SavedMirrorTemplate{
		Source:   m.source,
		Tag:      m.tag,
		ArchList: nil,
		Images:   nil,
	}
	for _, img := range m.images {
		if mT.ArchList == nil ||
			!slices.Contains(mT.ArchList, img.Arch()) {
			mT.ArchList = append(mT.ArchList, img.Arch())
		}
		iT := SavedImagesTemplate{
			Arch:    img.Arch(),
			OS:      img.OS(),
			Variant: img.Variant(),
			Digest:  img.Digest(),
		}
		mT.Images = append(mT.Images, iT)
	}
	if len(mT.ArchList) == 0 {
		return nil
	}

	return &mT
}

// LoadSavedTemplates loads the saved json templates to Mirrorer slice
func LoadSavedTemplates(directory, destReg string) ([]Mirrorer, error) {
	var err error
	if directory, err = u.GetAbsPath(directory); err != nil {
		return nil, fmt.Errorf("LoadSavedMirrorTemplate: %w", err)
	}
	logrus.Debugf("LoadSavedTemplates from dir: %v", directory)

	savedMirrorList := []SavedMirrorTemplate{}
	f, err := os.Open(filepath.Join(directory, u.SavedImageListFile))
	if err != nil {
		return nil, fmt.Errorf("LoadSavedMirrorTemplate: %w", err)
	}
	err = json.NewDecoder(f).Decode(&savedMirrorList)
	if err != nil {
		return nil, fmt.Errorf("LoadSavedMirrorTemplate: %w", err)
	}

	var mirrorerList []Mirrorer
	for _, mT := range savedMirrorList {
		m := NewMirror(&MirrorOptions{
			Source:      mT.Source,
			Destination: ConstructRegistry(mT.Source, destReg),
			Directory:   directory,
			Tag:         mT.Tag,
			ArchList:    mT.ArchList,
			Mode:        MODE_LOAD,
		})

		for _, iT := range mT.Images {
			copiedTag := image.CopiedTag(mT.Tag, iT.OS, iT.Arch, iT.Variant)
			// Source is a directory
			srcImage := fmt.Sprintf("%s:%s", mT.Source, copiedTag)
			srcImageDir := filepath.Join(directory, u.Sha256Sum(srcImage))
			// Destination is the dest registry
			repo := ConstructRegistry(mT.Source, destReg)
			destImage := fmt.Sprintf("%s:%s", repo, copiedTag)
			img := image.NewImage(&image.ImageOptions{
				Source:      srcImageDir,
				Destination: destImage,
				// Directory is the decompressed folder path
				Directory: directory,
				Tag:       mT.Tag,
				Arch:      iT.Arch,
				Variant:   iT.Variant,
				OS:        iT.OS,
				Digest:    iT.Digest,

				// saved image manifest is already converted to v2s2
				SourceSchemaVersion: 2,
				SourceMediaType:     u.MediaTypeManifestV2,
			})
			m.AppendImage(img)
		}
		mirrorerList = append(mirrorerList, m)
	}

	return mirrorerList, nil
}
