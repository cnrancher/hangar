package archive

import (
	"reflect"
	"time"
)

const (
	IndexVersion = "v1.2.0"
)

type HangarArchiveIndex struct {
	List      []*HangarArchiveImageList `json:"list,omitempty" yaml:"list,omitempty"`
	Version   string                    `json:"version,omitempty" yaml:"version.omitempty"`
	SavedTime time.Time                 `json:"time,omitempty" yaml:"omitempty"`
}

type HangarArchiveImageList struct {
	Source   string                   `json:"source,omitempty"`
	Tag      string                   `json:"tag,omitempty"`
	ArchList []string                 `json:"archList,omitempty"`
	OsList   []string                 `json:"osList,omitempty"`
	Images   []HangarArchiveImageSpec `json:"images,omitempty"`
}

type HangarArchiveImageSpec struct {
	Digest    string `json:"digest,omitempty"`
	Arch      string `json:"arch,omitempty"`
	OS        string `json:"os,omitempty"`
	OsVersion string `json:"osVersion,omitempty"`
	Variant   string `json:"variant,omitempty"`
	Folder    string `json:"folder,omitempty"`
}

// Deprecated: use HangarArchiveIndex instead
type SavedListTemplate HangarArchiveIndex

// Deprecated: use HangarArchiveIndex instead
type SavedMirrorTemplate HangarArchiveIndex

// Deprecated: use HangarArchiveImageSpec instead
type SavedImagesTemplate HangarArchiveImageSpec

func NewHangarArchiveIndex() *HangarArchiveIndex {
	return &HangarArchiveIndex{
		List:      make([]*HangarArchiveImageList, 0),
		Version:   IndexVersion,
		SavedTime: time.Now(),
	}
}

func (s *HangarArchiveIndex) Append(i *HangarArchiveImageList) {
	if i == nil {
		return
	}
	s.List = append(s.List, i)
}

func (s *HangarArchiveIndex) Has(i *HangarArchiveImageList) bool {
	for _, v := range s.List {
		if reflect.DeepEqual(v, i) {
			return true
		}
	}
	return false
}
