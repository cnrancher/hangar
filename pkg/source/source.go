package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/manifest"
	"github.com/cnrancher/hangar/pkg/types"
	imagemanifest "github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// Source represents the source image to be copied.
// The type of the source image can be:
// docker, docker-daemon, docker-archive, oci or dir
type Source struct {
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
	// digest
	digest digest.Digest

	// referenceName is the image reference with transport
	referenceName string

	// mime is the MIME type of image
	mime string

	// if mime is DockerV2ListMediaType
	schema2List *imagemanifest.Schema2List

	// if mime is DockerV2Schema2MediaType
	schema2 *imagemanifest.Schema2
	// if mime is DockerV2Schema2MediaType
	imageInspectInfo *imagetypes.ImageInspectInfo

	// if mime is DockerV2Schema1MediaType
	schema1 *imagemanifest.Schema1

	// if mime is DockerV2Schema2MediaType or DockerV2Schema1MediaType
	ociConfig *imgspecv1.Image

	// if mime is MediaTypeImageIndex
	ociIndex *imgspecv1.Index

	// if mime is MediaTypeImageManifest
	ociManifest *imgspecv1.Manifest

	// manifest digest
	manifestDigest digest.Digest

	systemCtx *imagetypes.SystemContext

	// copied image list
	copiedList []archive.ImageSpec

	// copied arch list
	copiedArch map[string]bool

	// copied OS list
	copiedOS map[string]bool
}

// Option is used for create the Source object.
type Option struct {
	// Image Type.
	Type types.ImageType
	// Directory, need to provide if Type is dir / oci / docker-archive
	Directory string
	// Registry, need to provide if Type is docker, docker-daemon, docker-archive
	Registry string
	// Project (also called namespace on some public cloud providers),
	// need to provide if Type is docker / docker-daemon / docker-archive
	Project string
	// Image name, need to provide if Type is docker / docker-daemon / docker-archive
	Name string
	// Image tag, need to provide if Type is docker / docker-daemon / docker-archive
	Tag string
	// Digest is used to identify the Digest of the image to be copied,
	// only available when Type is docker.
	Digest digest.Digest

	SystemContext *imagetypes.SystemContext
}

// NewSource is the constructor to create a Source object.
// Need to call Init method after creating the Source object before use.
func NewSource(o *Option) (*Source, error) {
	var (
		s   *Source
		err error
	)
	switch o.Type {
	case types.TypeDocker:
		s, err = newSourceFromDocker(o)
		if err != nil {
			return nil, err
		}
	case types.TypeDockerArhive:
		s, err = newSourceFromDockerArchive(o)
		if err != nil {
			return nil, err
		}
	case types.TypeDockerDaemon:
		s, err = newSourceFromDockerDaemon(o)
		if err != nil {
			return nil, err
		}
	case types.TypeOci:
		s, err = newSourceFromOci(o)
		if err != nil {
			return nil, err
		}
	case types.TypeDir:
		s, err = newSourceFromDir(o)
		if err != nil {
			return nil, err
		}
	default:
		return nil, types.ErrInvalidType
	}
	s.copiedArch = make(map[string]bool)
	s.copiedOS = make(map[string]bool)

	return s, nil
}

// Init initialize the source image manifest.
func (s *Source) Init(ctx context.Context) error {
	if err := s.initReferenceName(); err != nil {
		return err
	}
	return s.initManifest(ctx)
}

// Type returns the type of the image
func (s *Source) Type() types.ImageType {
	return s.imageType
}

func (s *Source) Directory() string {
	return s.directory
}

func (s *Source) Registry() string {
	return s.registry
}

func (s *Source) Project() string {
	return s.project
}

func (s *Source) Name() string {
	return s.name
}

func (s *Source) Tag() string {
	return s.tag
}

// ReferenceName returns the reference with transport of the source image.
//
//	Example:
//		docker://docker.io/library/hello-world:latest
//		docker-daemon://docker.io/library/nginx:1.23
//		oci:./path/to/oci-image
func (s *Source) ReferenceName() string {
	return s.referenceName
}

func (s *Source) Reference() (imagetypes.ImageReference, error) {
	return alltransports.ParseImageName(s.referenceName)
}

func (s *Source) ReferenceNameWithoutTransport() string {
	prefix := s.imageType.Transport()
	if prefix == "" {
		return ""
	}

	return strings.TrimPrefix(s.referenceName, prefix)
}

func (s *Source) ReferenceNameWithoutTransportAndTag() string {
	prefix := s.imageType.Transport()
	if prefix == "" {
		return ""
	}

	return strings.TrimPrefix(s.referenceName, prefix)
}

func (s *Source) MIME() string {
	return s.mime
}

func (s *Source) InspectRAW(ctx context.Context) ([]byte, string, error) {
	inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
		ReferenceName: s.referenceName,
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
	s.mime = mime
	return m, mime, err
}

func (s *Source) SystemContext() *imagetypes.SystemContext {
	return s.systemCtx
}

func (s *Source) Copy(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
) error {
	if dest.MIME() == imgspecv1.MediaTypeImageIndex ||
		dest.MIME() == imagemanifest.DockerV2ListMediaType {
		// TODO: destination image may exists
		_ = ""
	}

	switch s.mime {
	case imagemanifest.DockerV2ListMediaType:
		// manifest is docker image list
		num, err := s.copyDockerV2ListMediaType(ctx, dest, sets)
		if err != nil {
			return err
		}
		logrus.Debugf("copied [%d] images", num)
		if num == 0 {
			logrus.Warnf("no available image copied")
		}
	case imgspecv1.MediaTypeImageIndex:
		// manifest is oci image list
		num, err := s.copyMediaTypeImageIndex(ctx, dest, sets)
		if err != nil {
			return err
		}
		logrus.Debugf("copied [%d] images", num)
		if num == 0 {
			logrus.Warnf("no available image copied")
		}
	case imagemanifest.DockerV2Schema2MediaType:
		// manifest is docker image schema2
		err := s.copyDockerV2Schema2MediaType(ctx, dest, sets)
		if err != nil {
			return err
		}
	case imagemanifest.DockerV2Schema1MediaType,
		imagemanifest.DockerV2Schema1SignedMediaType:
		// manifest is docker image schema1
		err := s.copyDockerV2Schema1MediaType(ctx, dest, sets)
		if err != nil {
			return err
		}
	case imgspecv1.MediaTypeImageManifest:
		// manifest is oci image
		err := s.copyMediaTypeImageManifest(ctx, dest, sets)
		if err != nil {
			return err
		}
	}

	return nil
}

func newSourceFromDir(o *Option) (*Source, error) {
	if o.Type != types.TypeDir {
		return nil, types.ErrInvalidType
	}
	s := &Source{
		imageType: o.Type,
		directory: o.Directory,
		systemCtx: o.SystemContext,
	}

	return s, nil
}

func newSourceFromOci(o *Option) (*Source, error) {
	if o.Type != types.TypeOci {
		return nil, types.ErrInvalidType
	}
	s := &Source{
		imageType: o.Type,
		directory: o.Directory,
		tag:       o.Tag,
		systemCtx: o.SystemContext,
	}

	return s, nil
}

func newSourceFromDocker(o *Option) (*Source, error) {
	if o.Type != types.TypeDocker {
		return nil, types.ErrInvalidType
	}
	s := &Source{
		imageType: o.Type,
		registry:  o.Registry,
		project:   o.Project,
		name:      o.Name,
		tag:       o.Tag,
		systemCtx: o.SystemContext,
	}
	if s.tag == "" {
		if o.Digest != "" {
			s.digest = o.Digest
		} else {
			s.tag = "latest"
		}
	}
	if s.project == "" {
		s.project = "library"
	}
	if s.registry == "" {
		s.registry = "docker.io"
	}

	return s, nil
}

func newSourceFromDockerDaemon(o *Option) (*Source, error) {
	if o.Type != types.TypeDockerDaemon {
		return nil, types.ErrInvalidType
	}
	s := &Source{
		imageType: o.Type,
		registry:  o.Registry,
		project:   o.Project,
		name:      o.Name,
		tag:       o.Tag,
		systemCtx: o.SystemContext,
	}
	if s.tag == "" {
		s.tag = "latest"
	}
	if s.project == "" {
		s.project = "library"
	}
	if s.registry == "" {
		s.registry = "docker.io"
	}

	return s, nil
}

func newSourceFromDockerArchive(o *Option) (*Source, error) {
	if o.Type != types.TypeDockerArhive {
		return nil, types.ErrInvalidType
	}
	s := &Source{
		imageType: o.Type,
		directory: o.Directory,
		registry:  o.Registry,
		project:   o.Project,
		name:      o.Name,
		tag:       o.Tag,
		systemCtx: o.SystemContext,
	}
	if s.tag == "" {
		s.tag = "latest"
	}
	return s, nil
}

func (s *Source) initReferenceName() error {
	switch s.imageType {
	case types.TypeDocker:
		// docker://docker-reference
		if s.tag != "" {
			// example: docker://docker.io/library/nginx:1.23
			s.referenceName = fmt.Sprintf("%s%s/%s/%s:%s",
				s.imageType.Transport(),
				s.registry, s.project, s.name, s.tag)
		} else {
			// example: docker://docker.io/library/nginx@sha256:abcdef...
			s.referenceName = fmt.Sprintf("%s%s/%s/%s@%s",
				s.imageType.Transport(),
				s.registry, s.project, s.name, s.digest.String())
		}
	case types.TypeDockerArhive:
		// docker-archive:path[:docker-reference]
		// example: docker-archive:./path/to/tar:docker.io/library/nginx:1.23
		s.referenceName = fmt.Sprintf("%s%s:%s/%s/%s:%s",
			s.imageType.Transport(), s.directory,
			s.registry, s.project, s.name, s.tag)
	case types.TypeDockerDaemon:
		// docker-daemon:docker-reference
		// example: docker-daemon://docker.io/library/nginx:1.23
		s.referenceName = fmt.Sprintf("%s%s/%s/%s:%s",
			s.imageType.Transport(),
			s.registry, s.project, s.name, s.tag)
	case types.TypeDir:
		// dir:path
		// example: dir:path/to/image/
		s.referenceName = fmt.Sprintf("%s%s",
			s.imageType.Transport(), s.directory)
	case types.TypeOci:
		// oci:path:tag
		// example: oci:path/to/image:tag
		s.referenceName = fmt.Sprintf("%s%s",
			s.imageType.Transport(), s.directory)
	default:
		return types.ErrInvalidType
	}
	return nil
}

func (s *Source) initManifest(ctx context.Context) error {
	var err error
	inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
		ReferenceName: s.referenceName,
		SystemContext: s.systemCtx,
	})
	if err != nil {
		return err
	}
	defer inspector.Close()

	b, mime, err := inspector.Raw(ctx)
	if err != nil {
		return err
	}
	s.manifestDigest, err = imagemanifest.Digest(b)
	if err != nil {
		return err
	}

	// cache the source MIME
	s.mime = mime
	switch mime {
	// Docker image list
	case imagemanifest.DockerV2ListMediaType:
		s2list, err := imagemanifest.Schema2ListFromManifest(b)
		if err != nil {
			return err
		}
		s.schema2List = s2list
	// Docker image v2s2
	case imagemanifest.DockerV2Schema2MediaType:
		s2, err := imagemanifest.Schema2FromManifest(b)
		if err != nil {
			return err
		}
		s.schema2 = s2
		info, err := inspector.Inspect(ctx)
		if err != nil {
			return err
		}
		s.imageInspectInfo = info
		config, err := inspector.Config(ctx)
		if err != nil {
			return err
		}
		ociConfig := &imgspecv1.Image{}
		err = json.Unmarshal(config, ociConfig)
		if err != nil {
			return fmt.Errorf("initManifest: get ociConfig failed: %w", err)
		}
		s.ociConfig = ociConfig
	// Docker image v2s1
	case imagemanifest.DockerV2Schema1MediaType,
		imagemanifest.DockerV2Schema1SignedMediaType:
		s1, err := imagemanifest.Schema1FromManifest(b)
		if err != nil {
			return fmt.Errorf("initManifest: get ociManifest failed: %w", err)
		}
		info, err := inspector.Inspect(ctx)
		if err != nil {
			return fmt.Errorf("initManifest: %w", err)
		}
		s.schema1 = s1
		s.imageInspectInfo = info
	// OCI image list
	case imgspecv1.MediaTypeImageIndex:
		ociIndex := &imgspecv1.Index{}
		err = json.Unmarshal(b, ociIndex)
		if err != nil {
			return fmt.Errorf("initManifest: %w", err)
		}
		s.ociIndex = ociIndex
	// OCI image
	case imgspecv1.MediaTypeImageManifest:
		ociManifest := &imgspecv1.Manifest{}
		err = json.Unmarshal(b, ociManifest)
		if err != nil {
			return fmt.Errorf("initManifest: %w", err)
		}
		s.ociManifest = ociManifest
	default:
		return fmt.Errorf("unsupported MIME type %q", mime)
	}

	return nil
}
