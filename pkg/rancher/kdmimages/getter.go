package kdmimages

import (
	"context"
	"fmt"

	"github.com/rancher/rke/types/kdm"
)

type ClusterType string

const (
	K3S  ClusterType = "k3s"
	RKE2 ClusterType = "rke2"
	RKE  ClusterType = "rke"
)

func (t ClusterType) String() string {
	return string(t)
}

// Getter is the interface for getting images and versions from KDM data.
type Getter interface {
	// Get method is for getting the images and versions.
	Get(ctx context.Context) error

	// LinuxImageSet method gets the linux images and sources.
	LinuxImageSet() map[string]map[string]bool

	// WindowsImageSet method gets the Windows images and sources.
	WindowsImageSet() map[string]map[string]bool

	// VersionSet method gets the versions.
	VersionSet() map[string]bool

	// Source method gets the cluster type of getter.
	Source() ClusterType
}

type GetterOptions struct {
	Type           ClusterType
	RancherVersion string
	MinKubeVersion string
	KDMData        kdm.Data

	// RemoveDeprecated removes the old deprecated k8s versions and
	// only outputs the highest k8s patch version
	RemoveDeprecated bool
	InsecureSkipTLS  bool
}

func NewGetter(o *GetterOptions) (Getter, error) {
	switch o.Type {
	case K3S, RKE2:
		return newK3sRKE2Getter(o)
	case RKE:
		return newRKEGetter(o)
	}
	return nil, fmt.Errorf("unrecognized cluster type: %q", o.Type)
}
