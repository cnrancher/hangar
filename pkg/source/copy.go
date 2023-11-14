package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/copy"
	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/manifest"
	"github.com/containers/common/pkg/retry"
	imagecopy "github.com/containers/image/v5/copy"
	imagemanifest "github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func (s *Source) copyDockerV2ListMediaType(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
	policy *signature.Policy,
) (int, error) {
	var copiedNum int = 0
	var errs []error
	for _, m := range s.schema2List.Manifests {
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		osVersion := m.Platform.OSVersion
		variant := m.Platform.Variant
		dig := m.Digest
		mime := m.MediaType

		// skip image
		if len(sets["os"]) != 0 && osInfo != "" && !sets["os"][osInfo] {
			continue
		}
		if len(sets["arch"]) != 0 && arch != "" && !sets["arch"][arch] {
			continue
		}
		if len(sets["variant"]) != 0 && variant != "" && !sets["variant"][variant] {
			continue
		}

		sourceRef, err := alltransports.ParseImageName(fmt.Sprintf(
			"%s%s/%s/%s@%s",
			s.imageType.Transport(), s.registry, s.project, s.name, dig))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		destRef, err := dest.ReferenceMultiArch(
			osInfo, osVersion, arch, variant, dig.Encoded())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = copyImage(
			ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext(), policy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
			Reference:     destRef,
			SystemContext: dest.SystemContext(),
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("newInspector failed: %w", err))
			continue
		}
		b, imageMIME, err := inspector.Raw(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("inspector.Raw failed: %w", err))
			continue
		}
		manifestDigest, err := imagemanifest.Digest(b)
		spec := archive.ImageSpec{
			Arch:      arch,
			OS:        osInfo,
			OsVersion: osVersion,
			Variant:   variant,
			MediaType: mime,
			Layers:    nil,
			Config:    "",
			Digest:    manifestDigest,
		}
		switch imageMIME {
		case imagemanifest.DockerV2Schema2MediaType:
			schema2, err := imagemanifest.Schema2FromManifest(b)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			updateSpecDockerV2Schema2(&spec, schema2)
		case imagemanifest.DockerV2Schema1MediaType,
			imagemanifest.DockerV2Schema1SignedMediaType:
			schema1, err := imagemanifest.Schema1FromManifest(b)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			updateSpecDockerV2Schema1(&spec, schema1)
		case imgspecv1.MediaTypeImageManifest:
			ociManifest := new(imgspecv1.Manifest)
			if err = json.Unmarshal(b, ociManifest); err != nil {
				errs = append(errs, err)
				continue
			}
			updateSpecImageManifest(&spec, ociManifest)
		default:
			errs = append(errs, fmt.Errorf("copied image mime unknow: %v", imageMIME))
			continue
		}
		err = s.recordCopiedImage(spec)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		copiedNum++
	}

	if len(errs) > 0 {
		return copiedNum, fmt.Errorf(
			"error occurred when copy image [%v] => [%v]: %v",
			s.referenceName, dest.ReferenceName(), errs,
		)
	}
	return copiedNum, nil
}

func (s *Source) copyMediaTypeImageIndex(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
	policy *signature.Policy,
) (int, error) {
	var copiedNum int = 0
	var errs []error
	for _, m := range s.ociIndex.Manifests {
		mime := m.MediaType
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		osVersion := m.Platform.OSVersion
		variant := m.Platform.Variant
		dig := m.Digest

		// skip image
		if len(sets["os"]) != 0 && osInfo != "" && !sets["os"][osInfo] {
			continue
		}
		if len(sets["arch"]) != 0 && arch != "" && !sets["arch"][arch] {
			continue
		}
		if len(sets["variant"]) != 0 && variant != "" && !sets["variant"][variant] {
			continue
		}

		sourceRef, err := alltransports.ParseImageName(fmt.Sprintf(
			"%s%s/%s/%s@%s",
			s.imageType.Transport(), s.registry, s.project, s.name, dig))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		destRef, err := dest.ReferenceMultiArch(
			osInfo, osVersion, arch, variant, dig.Encoded())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = copyImage(
			ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext(), policy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
			Reference:     destRef,
			SystemContext: dest.SystemContext(),
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("newInspector failed: %w", err))
			continue
		}
		b, imageMIME, err := inspector.Raw(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("inspector.Raw failed: %w", err))
			continue
		}
		manifestDigest, err := imagemanifest.Digest(b)
		if err != nil {
			errs = append(errs, fmt.Errorf("imagemanifest.Digest failed: %w", err))
			continue
		}
		spec := archive.ImageSpec{
			Arch:      arch,
			OS:        osInfo,
			OsVersion: osVersion,
			Variant:   variant,
			MediaType: mime,
			Layers:    nil,
			Config:    "",
			Digest:    manifestDigest,
		}
		switch imageMIME {
		case imagemanifest.DockerV2Schema2MediaType:
			schema2, err := imagemanifest.Schema2FromManifest(b)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			updateSpecDockerV2Schema2(&spec, schema2)
		case imagemanifest.DockerV2Schema1MediaType,
			imagemanifest.DockerV2Schema1SignedMediaType:
			schema1, err := imagemanifest.Schema1FromManifest(b)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			updateSpecDockerV2Schema1(&spec, schema1)
		case imgspecv1.MediaTypeImageManifest:
			ociManifest := new(imgspecv1.Manifest)
			if err = json.Unmarshal(b, ociManifest); err != nil {
				errs = append(errs, err)
				continue
			}
			updateSpecImageManifest(&spec, ociManifest)
		default:
			errs = append(errs, fmt.Errorf("copied image mime unknow: %v", imageMIME))
			continue
		}
		err = s.recordCopiedImage(spec)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		copiedNum++
	}
	if len(errs) > 0 {
		b := strings.Builder{}
		for _, e := range errs {
			b.WriteString(fmt.Sprintf("%v\n", e))
		}
		return copiedNum, fmt.Errorf(
			"error occurred when copy image [%v] => [%v]: \n%s",
			s.referenceName, dest.ReferenceName(), b.String(),
		)
	}
	return copiedNum, nil
}

func (s *Source) copyDockerV2Schema2MediaType(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
	policy *signature.Policy,
) error {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	osVersion := s.ociConfig.OSVersion
	variant := s.ociConfig.Variant

	// skip image
	if len(sets["os"]) != 0 && osInfo != "" && !sets["os"][osInfo] {
		return nil
	}
	if len(sets["arch"]) != 0 && arch != "" && !sets["arch"][arch] {
		return nil
	}
	if len(sets["variant"]) != 0 && variant != "" && !sets["variant"][variant] {
		return nil
	}

	sourceRef, err := s.Reference()
	if err != nil {
		return err
	}
	destRef, err := dest.ReferenceMultiArch(
		osInfo, osVersion, arch, variant, s.manifestDigest.Encoded())
	if err != nil {
		return err
	}
	err = copyImage(
		ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext(), policy)
	if err != nil {
		return err
	}
	spec := archive.ImageSpec{
		Arch:      arch,
		OS:        osInfo,
		OsVersion: osVersion,
		Variant:   variant,
		MediaType: s.mime,
		Layers:    nil,
		Config:    s.schema2.ConfigDescriptor.Digest,
		Digest:    s.manifestDigest,
	}
	updateSpecDockerV2Schema2(&spec, s.schema2)
	return s.recordCopiedImage(spec)
}

func (s *Source) copyDockerV2Schema1MediaType(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
	policy *signature.Policy,
) error {
	arch := s.imageInspectInfo.Architecture
	osInfo := s.imageInspectInfo.Os
	osVersion := ""
	variant := s.imageInspectInfo.Variant

	// skip image
	if len(sets["os"]) != 0 && osInfo != "" && !sets["os"][osInfo] {
		return nil
	}
	if len(sets["arch"]) != 0 && arch != "" && !sets["arch"][arch] {
		return nil
	}
	if len(sets["variant"]) != 0 && variant != "" && !sets["variant"][variant] {
		return nil
	}

	sourceRef, err := s.Reference()
	if err != nil {
		return err
	}
	destRef, err := dest.ReferenceMultiArch(
		osInfo, osVersion, arch, variant, s.manifestDigest.Encoded())
	if err != nil {
		return err
	}
	err = copyImage(
		ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext(), policy)
	if err != nil {
		return err
	}
	spec := archive.ImageSpec{
		Arch:      arch,
		OS:        osInfo,
		OsVersion: osVersion,
		Variant:   variant,
		MediaType: s.mime,
		Layers:    nil,
		Config:    "", // Schema1 does not config
		Digest:    s.manifestDigest,
	}
	updateSpecDockerV2Schema1(&spec, s.schema1)
	return s.recordCopiedImage(spec)
}

func (s *Source) copyMediaTypeImageManifest(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
	policy *signature.Policy,
) error {
	arch := s.ociManifest.Config.Platform.Architecture
	osInfo := s.ociManifest.Config.Platform.OS
	osVersion := s.ociManifest.Config.Platform.OSVersion
	variant := s.ociManifest.Config.Platform.Variant

	// skip image
	if len(sets["os"]) != 0 && osInfo != "" && !sets["os"][osInfo] {
		return nil
	}
	if len(sets["arch"]) != 0 && arch != "" && !sets["arch"][arch] {
		return nil
	}
	if len(sets["variant"]) != 0 && variant != "" && !sets["variant"][variant] {
		return nil
	}

	sourceRef, err := s.Reference()
	if err != nil {
		return err
	}
	destRef, err := dest.ReferenceMultiArch(
		osInfo, osVersion, arch, variant, s.manifestDigest.Encoded())
	if err != nil {
		return err
	}
	err = copyImage(
		ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext(), policy)
	if err != nil {
		return err
	}
	spec := archive.ImageSpec{
		Arch:      arch,
		OS:        osInfo,
		OsVersion: osVersion,
		Variant:   variant,
		MediaType: s.mime,
		Layers:    nil,
		Config:    s.ociManifest.Config.Digest,
		Digest:    s.manifestDigest,
	}
	updateSpecImageManifest(&spec, s.ociManifest)
	return s.recordCopiedImage(spec)
}

func (s *Source) recordCopiedImage(image archive.ImageSpec) error {
	s.copiedList = append(s.copiedList, image)
	s.copiedArch[image.Arch] = true
	s.copiedOS[image.OS] = true
	return nil
}

func (s *Source) GetCopiedImage() *archive.Image {
	var (
		archies = make([]string, 0, len(s.copiedArch))
		oses    = make([]string, 0, len(s.copiedOS))
	)
	for a := range s.copiedArch {
		archies = append(archies, a)
	}
	for o := range s.copiedOS {
		oses = append(oses, o)
	}
	list := &archive.Image{
		Source:   fmt.Sprintf("%s/%s/%s", s.registry, s.project, s.name),
		Tag:      s.tag,
		ArchList: archies,
		OsList:   oses,
		Images:   s.copiedList,
	}
	return list
}

func copyImage(
	ctx context.Context,
	sourceRef imagetypes.ImageReference,
	destRef imagetypes.ImageReference,
	sourceCtx *imagetypes.SystemContext,
	destCtx *imagetypes.SystemContext,
	policy *signature.Policy,
) error {
	var err error
	copier := copy.NewCopier(&copy.CopierOption{
		Options: &imagecopy.Options{
			// Add sign here if needed
			ReportWriter:         nil,
			SourceCtx:            sourceCtx,
			DestinationCtx:       destCtx,
			ProgressInterval:     time.Second,
			PreserveDigests:      true,
			MaxParallelDownloads: 3,
		},
		RetryOptions: &retry.Options{
			MaxRetry: 3,
			Delay:    time.Second,
		},

		SourceRef: sourceRef,
		DestRef:   destRef,
		Policy:    policy,
	})
	_, err = copier.Copy(ctx)
	return err
}

func updateSpecDockerV2Schema2(
	spec *archive.ImageSpec, schema2 *imagemanifest.Schema2,
) *archive.ImageSpec {
	spec.Config = schema2.ConfigDescriptor.Digest
	for _, layer := range schema2.LayersDescriptors {
		if len(layer.URLs) != 0 {
			// The layer is from internet, ignore here.
			continue
		}
		spec.Layers = append(spec.Layers, layer.Digest)
	}
	return spec
}

func updateSpecDockerV2Schema1(
	spec *archive.ImageSpec, schema1 *imagemanifest.Schema1,
) {
	layerDigestSet := map[digest.Digest]bool{}
	for _, layer := range schema1.FSLayers {
		layerDigestSet[layer.BlobSum] = true
	}
	for layer := range layerDigestSet {
		spec.Layers = append(spec.Layers, layer)
	}
}

func updateSpecImageManifest(
	spec *archive.ImageSpec, ociManifest *imgspecv1.Manifest,
) {
	for _, layer := range ociManifest.Layers {
		if len(layer.URLs) != 0 {
			// The layer is from internet, ignore here.
			continue
		}
		spec.Layers = append(spec.Layers, layer.Digest)
	}
}
