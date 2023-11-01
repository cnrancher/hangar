package archive

import (
	"time"

	"github.com/opencontainers/go-digest"
)

const (
	IndexVersion = "v1.2.0"
)

type Index struct {
	List    []*Image  `json:"list,omitempty" yaml:"list,omitempty"`
	Version string    `json:"version,omitempty" yaml:"version.omitempty"`
	Time    time.Time `json:"time,omitempty" yaml:"omitempty"`

	digestSet map[digest.Digest]bool
}

type Image struct {
	Source   string      `json:"source,omitempty" yaml:"source,omitempty"`
	Tag      string      `json:"tag,omitempty" yaml:"tag,omitempty"`
	ArchList []string    `json:"archList,omitempty" yaml:"archList,omitempty"`
	OsList   []string    `json:"osList,omitempty" yaml:"osList,omitempty"`
	Images   []ImageSpec `json:"images,omitempty" yaml:"images,omitempty"`
}

type ImageSpec struct {
	Arch      string          `json:"arch,omitempty" yaml:"arch,omitempty"`
	OS        string          `json:"os,omitempty" yaml:"os,omitempty"`
	OsVersion string          `json:"osVersion,omitempty" yaml:"osVersion,omitempty"`
	Variant   string          `json:"variant,omitempty" yaml:"variant,omitempty"`
	Folder    string          `json:"folder,omitempty" yaml:"folder,omitempty"`
	MediaType string          `json:"mediaType,omitempty" yaml:"mime,omitempty"`
	Layers    []digest.Digest `json:"layers,omitempty" yaml:"layers,omitempty"`
	Config    digest.Digest   `json:"config,omitempty" yaml:"config,omitempty"`
	Digest    digest.Digest   `json:"manifest,omitempty" yaml:"manifest,omitempty"`
}

func NewIndex() *Index {
	return &Index{
		List:      make([]*Image, 0),
		Version:   IndexVersion,
		Time:      time.Now(),
		digestSet: make(map[digest.Digest]bool),
	}
}

func (s *Index) Append(i *Image) {
	if i == nil {
		return
	}
	if s.Has(i) {
		return
	}
	if s.digestSet == nil {
		s.digestSet = make(map[digest.Digest]bool)
	}
	for _, img := range i.Images {
		s.digestSet[img.Digest] = true
	}
	s.List = append(s.List, i)
}

func (s *Index) Has(i *Image) bool {
	if s.digestSet == nil {
		return false
	}
	for _, img := range i.Images {
		if _, ok := s.digestSet[img.Digest]; !ok {
			return false
		}
	}
	return true
}
