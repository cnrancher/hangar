package types

import (
	"errors"
)

// ImageType represents some image types supported by Hangar.
type ImageType int

const (
	TypeUndefined ImageType = iota
	TypeDocker
	TypeDockerDaemon
	TypeDockerArhive
	TypeOci
	TypeDir
	TypeHangarArchive
)

var (
	ErrInvalidType = errors.New("invalid image type")
)

func (t ImageType) String() string {
	switch t {
	case TypeDocker:
		return "docker"
	case TypeDockerDaemon:
		return "docker-daemon"
	case TypeDockerArhive:
		return "docker-archive"
	case TypeOci:
		return "oci"
	case TypeDir:
		return "dir"
	default:
		return "undefined"
	}
}

func (t *ImageType) Transport() string {
	switch *t {
	case TypeDocker:
		return "docker://"
	case TypeDockerDaemon:
		return "docker-daemon://"
	case TypeDockerArhive:
		return "docker-archive:"
	case TypeOci:
		return "oci:"
	case TypeDir:
		return "dir:"
	default:
		return ""
	}
}

// FilterSet is a set to filter image arch, os, variants.
type FilterSet map[string]map[string]bool

// NewImageFilterSet is the constructor function to build a filter set
// by arch, os and variant list.
func NewImageFilterSet(archList, osList, variantList []string) FilterSet {
	s := FilterSet{
		"arch":    make(map[string]bool),
		"os":      make(map[string]bool),
		"variant": make(map[string]bool),
	}
	if len(archList) > 0 {
		for _, v := range archList {
			s["arch"][v] = true
		}
	}
	if len(osList) > 0 {
		for _, v := range osList {
			s["os"][v] = true
		}
	}
	if len(variantList) > 0 {
		for _, v := range variantList {
			s["variant"][v] = true
		}
	}
	return s
}

func (s FilterSet) Allow(arch, os, variant string) bool {
	return s.AllowArch(arch) && s.AllowOS(os) && s.AllowVariant(variant)
}

func (s FilterSet) AllowArch(arch string) bool {
	if len(s["arch"]) == 0 {
		return true
	}
	if len(arch) > 0 {
		return s["arch"][arch]
	}
	return false
}

func (s FilterSet) AllowOS(os string) bool {
	if len(s["os"]) == 0 {
		return true
	}
	if len(os) > 0 {
		return s["os"][os]
	}
	return false
}

func (s FilterSet) AllowVariant(v string) bool {
	if len(s["variant"]) == 0 {
		return true
	}
	if len(v) == 0 {
		return true
	}
	if len(v) > 0 {
		return s["variant"][v]
	}
	return false
}
