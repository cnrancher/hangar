package mirror

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	mTemplate := SavedMirrorTemplate{
		Source:   m.source,
		Tag:      m.tag,
		ArchList: nil,
		Images:   nil,
	}
	for _, img := range m.images {
		if mTemplate.ArchList == nil ||
			!slices.Contains(mTemplate.ArchList, img.Arch()) {
			mTemplate.ArchList = append(mTemplate.ArchList, img.Arch())
		}
		imgTemplate := SavedImagesTemplate{
			Arch:    img.Arch(),
			OS:      img.OS(),
			Variant: img.Variant(),
			Digest:  img.Digest(),
		}
		mTemplate.Images = append(mTemplate.Images, imgTemplate)
	}
	if len(mTemplate.ArchList) == 0 {
		return nil
	}

	return &mTemplate
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
	for _, mTemplate := range savedMirrorList {
		m := NewMirror(&MirrorOptions{
			Source:      mTemplate.Source,
			Destination: ConstructRegistry(mTemplate.Source, destReg),
			Directory:   directory,
			Tag:         mTemplate.Tag,
			ArchList:    mTemplate.ArchList,
			Mode:        MODE_LOAD,
		})

		for _, imgTemplate := range mTemplate.Images {
			// Source is a directory
			sourceImage := filepath.Join(directory,
				strings.TrimLeft(imgTemplate.Digest, "sha256:"))
			repo := ConstructRegistry(mTemplate.Source, destReg)
			tag := image.CopiedTag(
				mTemplate.Tag, imgTemplate.Arch, imgTemplate.Variant)
			// Destination is the dest registry
			destImage := fmt.Sprintf("%s:%s", repo, tag)
			img := image.NewImage(&image.ImageOptions{
				Source:      sourceImage,
				Destination: destImage,
				// Directory is the decompressed folder path
				Directory: directory,
				Tag:       mTemplate.Tag,
				Arch:      imgTemplate.Arch,
				Variant:   imgTemplate.Variant,
				OS:        imgTemplate.OS,
				Digest:    imgTemplate.Digest,

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
