package utils

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

var (
	ErrReadJsonFailed       = errors.New("failed to read value from json")
	ErrSkopeoNotFound       = errors.New("skopeo not found")
	ErrDockerNotFound       = errors.New("docker not found")
	ErrLoginFailed          = errors.New("login failed")
	ErrNoAvailableImage     = errors.New("no image available for specified arch list")
	ErrInvalidParameter     = errors.New("invalid parameter")
	ErrInvalidMediaType     = errors.New("invalid media type")
	ErrInvalidSchemaVersion = errors.New("invalid schema version")
	ErrNilPointer           = errors.New("nil pointer")
	ErrDockerBuildxNotFound = errors.New("docker buildx not found")
)

const (
	DockerLoginURL          = "https://hub.docker.com/v2/users/login/"
	DockerHubRegistry       = "docker.io"
	MediaTypeManifestListV2 = "application/vnd.docker.distribution.manifest.list.v2+json"
	MediaTypeManifestV2     = "application/vnd.docker.distribution.manifest.v2+json"
)

var (
	// worker number of mirrorer
	MirrorerJobNum = 1
)

func Sha256Sum(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}
