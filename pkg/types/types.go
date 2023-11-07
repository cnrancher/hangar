package types

import "errors"

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

func (t *ImageType) String() string {
	if t == nil {
		return "<nil>"
	}
	switch *t {
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
