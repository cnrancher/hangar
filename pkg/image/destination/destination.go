package destination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/image/manifest"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/opencontainers/go-digest"

	manifestv5 "github.com/containers/image/v5/manifest"
	alltransportsv5 "github.com/containers/image/v5/transports/alltransports"
	typesv5 "github.com/containers/image/v5/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Destination represents the destination of the image to be copied。
// The type of the destination image can be:
// docker, docker-daemon, oci or dir
// (docker-archive won't be supported by hangar)
type Destination struct {
	// imageType
	imageType types.ImageType

	// directory
	directory string
	// registry
	registry string
	// project (namespace)
	project string
	// image name
	name string
	// tag
	tag string

	// referenceName is the image reference with transport
	referenceName string

	// mime is the MIME type of image.
	// the mime will be empty string if destination image does not exists.
	mime string

	// if mime is DockerV2ListMediaType
	schema2List *manifestv5.Schema2List

	// if mime is MediaTypeImageIndex
	ociIndex *imgspecv1.Index

	systemCtx *typesv5.SystemContext
}

// Option is used for create the Destination object.
type Option struct {
	// Image Type.
	Type types.ImageType
	// Directory, need to provide if Type is dir / oci
	Directory string
	// Registry, need to provide if Type is docker / docker-daemon
	Registry string
	// Project (also called namespace on some public cloud providers),
	// need to provide if Type is docker / docker-daemon
	Project string
	// Image Name, need to provide if Type is docker / docker-daemon
	Name string
	// Image Tag, need to provide if Type is docker / docker-daemon
	Tag string

	SystemContext *typesv5.SystemContext
}

// NewDestination is the constructor to create a Destination object.
func NewDestination(o *Option) (*Destination, error) {
	var (
		d   *Destination
		err error
	)
	switch o.Type {
	case types.TypeDocker:
		d, err = newDestinationFromDocker(o)
		if err != nil {
			return nil, err
		}
	case types.TypeDockerDaemon:
		d, err = newDestinationFromDockerDaemon(o)
		if err != nil {
			return nil, err
		}
	case types.TypeOci:
		d, err = newDestinationFromOci(o)
		if err != nil {
			return nil, err
		}
	case types.TypeDir:
		d, err = newDestinationFromDir(o)
		if err != nil {
			return nil, err
		}
	default:
		return nil, types.ErrInvalidType
	}

	return d, nil
}

func (d *Destination) Init(ctx context.Context) error {
	err := d.initReferenceName()
	if err != nil {
		return err
	}
	// Ignore other error
	if err = d.initManifest(ctx); err != nil {
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) ||
			strings.Contains(err.Error(), "timeout") {
			return err
		}
	}
	return nil
}

// Type returns the type of the image
func (d *Destination) Type() types.ImageType {
	return d.imageType
}

func (d *Destination) Directory() string {
	return d.directory
}

// ReferenceName returns the reference name with transport of the source image.
//
//	Example:
//		docker://docker.io/library/hello-world:latest
//		docker-daemon://docker.io/library/nginx:1.23
//		oci:./path/to/oci-image
func (d *Destination) ReferenceName() string {
	return d.referenceName
}

func (d *Destination) MultiArchTag(os, osVersion, arch, variant string) string {
	if osVersion != "" {
		return fmt.Sprintf("%s-%s-%s-%s%s",
			d.referenceName, os, osVersion, arch, variant)
	}
	return fmt.Sprintf("%s-%s-%s%s", d.referenceName, os, arch, variant)
}

// ReferenceName returns the multi-arch (os, variant) reference name
// with transport of the source image.
//
//	Example:
//		docker://docker.io/library/hello-world:latest-linux-amd64
//		docker://docker.io/library/example:latest-windows-10.0.14393.1066-amd64
//		docker-daemon://docker.io/library/nginx:1.23-linux-arm64
//		oci:./path/to/oci-image/<sha256sum>
//		dir:./path/to/image/<sha256sum>
func (d *Destination) ReferenceNameMultiArch(
	os, osVersion, arch, variant, sha256sum string,
) string {
	switch d.imageType {
	case types.TypeDir,
		types.TypeOci:
		return filepath.Join(d.referenceName, sha256sum)
	default:
		if arch == "unknown" {
			return fmt.Sprintf("%s-%s", d.referenceName, sha256sum[:16])
		}
		return d.MultiArchTag(os, osVersion, arch, variant)
	}
}

func (d *Destination) Reference() (typesv5.ImageReference, error) {
	return alltransportsv5.ParseImageName(d.referenceName)
}

func (d *Destination) ReferenceMultiArch(
	os, osVersion, arch, variant, sha256sum string,
) (typesv5.ImageReference, error) {
	refName := d.ReferenceNameMultiArch(os, osVersion, arch, variant, sha256sum)
	return alltransportsv5.ParseImageName(refName)
}

func (d *Destination) ReferenceNameWithoutTransport() string {
	prefix := d.imageType.Transport()
	if prefix == "" {
		return ""
	}

	return strings.TrimPrefix(d.referenceName, prefix)
}

func (d *Destination) ReferenceNameDigest(dig digest.Digest) string {
	return fmt.Sprintf("%s@%s",
		strings.TrimSuffix(d.referenceName, ":"+d.tag), dig.String())
}

func (d *Destination) MIME() string {
	return d.mime
}

func (d *Destination) InspectRAW(ctx context.Context) ([]byte, string, error) {
	inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
		ReferenceName: d.referenceName,
	})
	if err != nil {
		return nil, "", fmt.Errorf("newInspector: %w", err)
	}
	defer inspector.Close()

	m, mime, err := inspector.Raw(ctx)
	if err != nil {
		return nil, "", err
	}
	// Refresh the cached MIME.
	d.mime = mime
	return m, mime, err
}

// Exists checks the destination image is exists or not.
func (d *Destination) Exists() bool {
	return d.mime != ""
}

func (d *Destination) SystemContext() *typesv5.SystemContext {
	return d.systemCtx
}

func (d *Destination) initReferenceName() error {
	switch d.imageType {
	case types.TypeDocker:
		// docker://docker-reference
		// example: docker://docker.io/library/nginx:1.23
		d.referenceName = fmt.Sprintf("%s%s/%s/%s:%s",
			d.imageType.Transport(),
			d.registry, d.project, d.name, d.tag)
	case types.TypeDockerDaemon:
		// docker-daemon:docker-reference
		// example: docker-daemon://docker.io/library/nginx:1.23
		d.referenceName = fmt.Sprintf("%s%s/%s/%s:%s",
			d.imageType.Transport(),
			d.registry, d.project, d.name, d.tag)
	case types.TypeDir:
		// dir:path
		// example: dir:path/to/image/
		d.referenceName = fmt.Sprintf("%s%s",
			d.imageType.Transport(), d.directory)
	case types.TypeOci:
		// oci:path:tag
		// example: oci:path/to/image:tag
		d.referenceName = fmt.Sprintf("%s%s",
			d.imageType.Transport(), d.directory)
	default:
		return types.ErrInvalidType
	}
	return nil
}

func (d *Destination) initManifest(ctx context.Context) error {
	var err error
	inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
		ReferenceName: d.referenceName,
		SystemContext: d.systemCtx,
	})
	if err != nil {
		return err
	}
	defer inspector.Close()

	b, mime, err := inspector.Raw(ctx)
	if err != nil {
		return err
	}

	// cache the destination MIME
	d.mime = mime

	// Only record DockerV2ListMediaType and MediaTypeImageIndex here
	// since the destination image on registry server should be managed
	// by DockerV2ListMediaType or MediaTypeImageIndex.
	switch mime {
	// Docker image list
	case manifestv5.DockerV2ListMediaType:
		s2list, err := manifestv5.Schema2ListFromManifest(b)
		if err != nil {
			return err
		}
		d.schema2List = s2list
	// OCI image list
	case imgspecv1.MediaTypeImageIndex:
		ociIndex := &imgspecv1.Index{}
		err = json.Unmarshal(b, ociIndex)
		if err != nil {
			return fmt.Errorf("initManifest: %w", err)
		}
		d.ociIndex = ociIndex
	}

	return nil
}

func newDestinationFromDir(o *Option) (*Destination, error) {
	if o.Type != types.TypeDir {
		return nil, types.ErrInvalidType
	}
	d := &Destination{
		imageType: o.Type,
		directory: o.Directory,
		systemCtx: o.SystemContext,
		tag:       o.Tag,
	}

	return d, nil
}

func newDestinationFromOci(o *Option) (*Destination, error) {
	if o.Type != types.TypeOci {
		return nil, types.ErrInvalidType
	}
	d := &Destination{
		imageType: o.Type,
		directory: o.Directory,
		systemCtx: o.SystemContext,
		tag:       o.Tag,
	}

	return d, nil
}

func newDestinationFromDocker(o *Option) (*Destination, error) {
	if o.Type != types.TypeDocker {
		return nil, types.ErrInvalidType
	}
	d := &Destination{
		imageType: o.Type,
		registry:  o.Registry,
		project:   o.Project,
		name:      o.Name,
		tag:       o.Tag,
		systemCtx: o.SystemContext,
	}
	if d.tag == "" {
		d.tag = utils.DefaultTag
	}
	if d.project == "" {
		d.project = utils.DefaultProject
	}
	if d.registry == "" {
		d.registry = utils.DockerHubRegistry
	}

	return d, nil
}

func newDestinationFromDockerDaemon(o *Option) (*Destination, error) {
	if o.Type != types.TypeDockerDaemon {
		return nil, types.ErrInvalidType
	}
	d := &Destination{
		imageType: o.Type,
		registry:  o.Registry,
		project:   o.Project,
		name:      o.Name,
		tag:       o.Tag,
		systemCtx: o.SystemContext,
	}
	if d.tag == "" {
		d.tag = utils.DefaultTag
	}
	if d.project == "" {
		d.project = utils.DefaultProject
	}
	if d.registry == "" {
		d.registry = utils.DockerHubRegistry
	}

	return d, nil
}

func (d *Destination) ImageBySet(set types.FilterSet) *archive.Image {
	image := &archive.Image{}
	if !d.Exists() {
		return image
	}
	archSet := map[string]bool{}
	osSet := map[string]bool{}
	switch d.mime {
	case manifestv5.DockerV2ListMediaType:
		for _, m := range d.schema2List.Manifests {
			p := &m.Platform
			if !set.Allow(p.Architecture, p.OS, p.Variant) {
				continue
			}
			archSet[p.Architecture] = true
			osSet[p.OS] = true
			image.Images = append(image.Images, archive.ImageSpec{
				Digest: m.Digest,
			})
		}
	case imgspecv1.MediaTypeImageIndex:
		for _, m := range d.ociIndex.Manifests {
			p := m.Platform
			if !set.Allow(p.Architecture, p.OS, p.Variant) {
				continue
			}
			archSet[p.Architecture] = true
			osSet[p.OS] = true
			image.Images = append(image.Images, archive.ImageSpec{
				Digest: m.Digest,
			})
		}
	}
	for arch := range archSet {
		image.ArchList = append(image.ArchList, arch)
	}
	for os := range osSet {
		image.OsList = append(image.OsList, os)
	}
	return image
}

func (d *Destination) ManifestImages() manifest.Images {
	var mis manifest.Images
	switch d.mime {
	case manifestv5.DockerV2ListMediaType:
		for _, m := range d.schema2List.Manifests {
			mi := manifest.NewImage(m.Digest, m.MediaType, m.Size, nil)
			mi.UpdatePlatform(
				m.Platform.Architecture,
				m.Platform.Variant,
				m.Platform.OS,
				m.Platform.OSVersion,
				m.Platform.OSFeatures,
			)
			mis = append(mis, mi)
		}
	case imgspecv1.MediaTypeImageIndex:
		for _, m := range d.ociIndex.Manifests {
			mi := manifest.NewImage(m.Digest, m.MediaType, m.Size, m.Annotations)
			mi.UpdatePlatform(
				m.Platform.Architecture,
				m.Platform.Variant,
				m.Platform.OS,
				m.Platform.OSVersion,
				m.Platform.OSFeatures,
			)
			mis = append(mis, mi)
		}
	}
	return mis
}

func (d *Destination) HaveDigest(imageDigest digest.Digest) bool {
	if d.mime == "" || imageDigest == "" {
		return false
	}

	switch d.mime {
	case manifestv5.DockerV2ListMediaType:
		for _, m := range d.schema2List.Manifests {
			if m.Digest == imageDigest {
				return true
			}
		}
	case imgspecv1.MediaTypeImageIndex:
		for _, m := range d.ociIndex.Manifests {
			if m.Digest == imageDigest {
				return true
			}
		}
	}
	return false
}
