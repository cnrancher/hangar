package source

import (
	"context"
	"fmt"
	"time"

	"github.com/cnrancher/hangar/pkg/copy"
	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/containers/common/pkg/retry"
	imagecopy "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/transports/alltransports"
	imagetypes "github.com/containers/image/v5/types"
)

func (s *Source) copyDockerV2ListMediaType(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
) (int, error) {
	var copiedNum int = 0
	var errs []error
	for _, m := range s.schema2List.Manifests {
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
		destRef, err := dest.ReferenceMultiArch(osInfo, osVersion, arch, variant)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = copyImage(ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		copiedNum++
	}

	if len(errs) > 0 {
		return copiedNum, fmt.Errorf(
			"error occured when copy image [%v] => [%v]: %v",
			s.referenceName, dest.ReferenceName(), errs,
		)
	}
	return copiedNum, nil
}

func (s *Source) copyMediaTypeImageIndex(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
) (int, error) {
	var copiedNum int = 0
	var errs []error
	for _, m := range s.ociIndex.Manifests {
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
		destRef, err := dest.ReferenceMultiArch(osInfo, osVersion, arch, variant)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = copyImage(ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		copiedNum++
	}
	if len(errs) > 0 {
		return copiedNum, fmt.Errorf(
			"error occured when copy image [%v] => [%v]: %v",
			s.referenceName, dest.ReferenceName(), errs,
		)
	}
	return copiedNum, nil
}

func (s *Source) copyDockerV2Schema2MediaType(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
) error {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
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
	destRef, err := dest.Reference()
	if err != nil {
		return err
	}
	return copyImage(ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext())
}

func (s *Source) copyDockerV2Schema1MediaType(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
) error {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
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
	destRef, err := dest.Reference()
	if err != nil {
		return err
	}
	return copyImage(ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext())
}

func (s *Source) copyMediaTypeImageManifest(
	ctx context.Context,
	dest *destination.Destination,
	sets map[string]map[string]bool,
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
	destRef, err := dest.ReferenceMultiArch(osInfo, osVersion, arch, variant)
	if err != nil {
		return err
	}
	return copyImage(ctx, sourceRef, destRef, s.systemCtx, dest.SystemContext())
}

func copyImage(
	ctx context.Context,
	sourceRef imagetypes.ImageReference,
	destRef imagetypes.ImageReference,
	sourceCtx *imagetypes.SystemContext,
	destCtx *imagetypes.SystemContext,
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
	})
	_, err = copier.Copy(ctx)
	return err
}
