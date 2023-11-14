package archive

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/opencontainers/go-digest"
)

const (
	IndexVersion = "v1.2.0"
)

// Index defines the data structure stores in the end of hangar archive.
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
	MediaType string          `json:"mediaType,omitempty" yaml:"mime,omitempty"`
	Layers    []digest.Digest `json:"layers,omitempty" yaml:"layers,omitempty"`
	Config    digest.Digest   `json:"config,omitempty" yaml:"config,omitempty"`
	Digest    digest.Digest   `json:"digest,omitempty" yaml:"digest,omitempty"`
}

func NewIndex() *Index {
	return &Index{
		List:      make([]*Image, 0),
		Version:   IndexVersion,
		Time:      time.Now(),
		digestSet: make(map[digest.Digest]bool),
	}
}

func UnmarshalIndex(b []byte) (*Index, error) {
	i := &Index{}
	err := json.Unmarshal(b, i)
	if err != nil {
		return nil, fmt.Errorf("UnmarshalIndex: %w", err)
	}
	i.digestSet = make(map[digest.Digest]bool)
	for _, images := range i.List {
		for _, image := range images.Images {
			i.digestSet[image.Digest] = true
		}
	}
	return i, nil
}

func (i *Index) Unmarshal(b []byte) error {
	err := json.Unmarshal(b, i)
	if err != nil {
		return err
	}
	i.digestSet = make(map[digest.Digest]bool)
	for _, images := range i.List {
		for _, image := range images.Images {
			i.digestSet[image.Digest] = true
		}
	}
	return nil
}

func (s *Index) Append(i *Image) {
	if i == nil {
		return
	}
	// if s.Has(i) {
	// 	return
	// }
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
