package archive

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cnrancher/hangar/pkg/utils"
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
	Arch       string          `json:"arch,omitempty" yaml:"arch,omitempty"`
	OS         string          `json:"os,omitempty" yaml:"os,omitempty"`
	OSVersion  string          `json:"osVersion,omitempty" yaml:"osVersion,omitempty"`
	OSFeatures []string        `json:"osFeatures,omitempty" yaml:"osFeatures,omitempty"`
	Variant    string          `json:"variant,omitempty" yaml:"variant,omitempty"`
	MediaType  string          `json:"mediaType,omitempty" yaml:"mime,omitempty"`
	Layers     []digest.Digest `json:"layers,omitempty" yaml:"layers,omitempty"`
	Config     digest.Digest   `json:"config,omitempty" yaml:"config,omitempty"`
	Digest     digest.Digest   `json:"digest,omitempty" yaml:"digest,omitempty"`
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

func (i *Index) Append(n *Image) {
	if n == nil {
		return
	}
	if len(n.Images) == 0 {
		return
	}
	// if i.Has(n) {
	// 	return
	// }
	if i.digestSet == nil {
		i.digestSet = make(map[digest.Digest]bool)
	}
	for _, img := range n.Images {
		i.digestSet[img.Digest] = true
	}
	i.List = append(i.List, n)
}

func (i *Index) Has(n *Image) bool {
	if i.digestSet == nil {
		return false
	}
	for _, img := range n.Images {
		if _, ok := i.digestSet[img.Digest]; !ok {
			return false
		}
	}
	return true
}

func (i *Index) HasReference(project, name, tag string) bool {
	for _, images := range i.List {
		p := utils.GetProjectName(images.Source)
		n := utils.GetImageName(images.Source)
		t := images.Tag
		if p == project && n == name && t == tag {
			return true
		}
	}
	return false
}
