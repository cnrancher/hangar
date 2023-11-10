package destination

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/manifest"
	"github.com/cnrancher/hangar/pkg/types"
	"github.com/cnrancher/hangar/pkg/utils"
	imagemanifest "github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
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
	// multi-arch hash tag
	multiArchHashTag string

	// referenceName is the image reference with transport
	referenceName string

	// mime is the MIME type of image.
	// the mime will be empty string if destination image does not exists.
	mime string

	// if mime is DockerV2ListMediaType
	schema2List *imagemanifest.Schema2List

	// if mime is MediaTypeImageIndex
	ociIndex *imgspecv1.Index

	systemCtx *imagetypes.SystemContext
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

	SystemContext *imagetypes.SystemContext
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
	// Ignore error
	d.initManifest(ctx)
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

func (d *Destination) MultiArchHashTag(os, osVersion, arch, variant string) string {
	if osVersion != "" {
		return utils.Sha256Sum(utils.Sha256Sum(fmt.Sprintf("%s-%s-%s-%s%s",
			d.tag, os, osVersion, arch, variant)))
	} else {
		return utils.Sha256Sum(fmt.Sprintf("%s-%s-%s%s",
			d.tag, os, arch, variant))
	}
}

func (d *Destination) MultiArchTag(os, osVersion, arch, variant string) string {
	if osVersion != "" {
		return fmt.Sprintf("%s-%s-%s-%s%s",
			d.referenceName, os, osVersion, arch, variant)
	} else {
		return fmt.Sprintf("%s-%s-%s%s", d.referenceName, os, arch, variant)
	}
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
	os, osVersion, arch, variant string,
) string {
	switch d.imageType {
	case types.TypeDir,
		types.TypeOci:
		return path.Join(
			d.referenceName,
			d.MultiArchHashTag(os, osVersion, arch, variant))
	default:
		return d.MultiArchTag(os, osVersion, arch, variant)
	}
}

func (d *Destination) Reference() (imagetypes.ImageReference, error) {
	return alltransports.ParseImageName(d.referenceName)
}

func (d *Destination) ReferenceMultiArch(
	os, osVersion, arch, variant string,
) (imagetypes.ImageReference, error) {
	refName := d.ReferenceNameMultiArch(os, osVersion, arch, variant)
	return alltransports.ParseImageName(refName)
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

func (d *Destination) SystemContext() *imagetypes.SystemContext {
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
	case imagemanifest.DockerV2ListMediaType:
		s2list, err := imagemanifest.Schema2ListFromManifest(b)
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
		d.tag = "latest"
	}
	if d.project == "" {
		d.project = "library"
	}
	if d.registry == "" {
		d.registry = "docker.io"
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
		d.tag = "latest"
	}
	if d.project == "" {
		d.project = "library"
	}
	if d.registry == "" {
		d.registry = "docker.io"
	}

	return d, nil
}

func (d *Destination) ImageBySet(set map[string]map[string]bool) *archive.Image {
	image := &archive.Image{}
	if !d.Exists() {
		return image
	}
	archSet := map[string]bool{}
	osSet := map[string]bool{}
	switch d.mime {
	case imagemanifest.DockerV2ListMediaType:
		for _, m := range d.schema2List.Manifests {
			p := &m.Platform
			if len(set["arch"]) != 0 && !set["arch"][p.Architecture] {
				continue
			}
			if len(set["os"]) != 0 && !set["os"][p.OS] {
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
			if len(set["arch"]) != 0 && !set["arch"][p.Architecture] {
				continue
			}
			if len(set["os"]) != 0 && !set["os"][p.OS] {
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

func (d *Destination) ManifestBuilder(ctx context.Context) (*manifest.Builder, error) {
	builder, err := manifest.NewBuilder(&manifest.BuilderOpts{
		ReferenceName: d.ReferenceName(),
		SystemContext: d.systemCtx,
	})
	if err != nil {
		return nil, err
	}
	if d.mime == "" {
		return builder, nil
	}

	// Add existing images into manifest builder
	switch d.mime {
	case imagemanifest.DockerV2ListMediaType:
		for _, m := range d.schema2List.Manifests {
			rn := d.ReferenceNameDigest(m.Digest)
			mi, err := manifest.NewManifestImage(ctx, rn, d.systemCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to create manifest image: %w", err)
			}
			builder.Add(mi)
		}
	case imgspecv1.MediaTypeImageIndex:
		for _, m := range d.ociIndex.Manifests {
			rn := d.ReferenceNameDigest(m.Digest)
			mi, err := manifest.NewManifestImage(ctx, rn, d.systemCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to create manifest image: %w", err)
			}
			builder.Add(mi)
		}
	}

	return builder, nil
}
