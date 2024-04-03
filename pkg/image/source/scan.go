package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/cnrancher/hangar/pkg/image/scan"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"

	manifestv5 "github.com/containers/image/v5/manifest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

type ScanOptions struct {
	Set types.FilterSet
}

func (s *Source) Scan(
	ctx context.Context,
	opts *ScanOptions,
) (*scan.Result, error) {
	switch s.mime {
	case manifestv5.DockerV2ListMediaType:
		// manifest is docker image list
		result, num, err := s.scanDockerV2ListMediaType(ctx, opts)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("scanned [%d] images", num)
		if num == 0 {
			return nil, utils.ErrNoAvailableImage
		}
		return result, nil
	case imgspecv1.MediaTypeImageIndex:
		// manifest is oci image list
		result, num, err := s.scanMediaTypeImageIndex(ctx, opts)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("scanned [%d] images", num)
		if num == 0 {
			return nil, utils.ErrNoAvailableImage
		}
		return result, nil
	case manifestv5.DockerV2Schema2MediaType:
		// manifest is docker image schema2
		result, err := s.scanDockerV2Schema2MediaType(ctx, opts)
		if err != nil {
			return nil, err
		}
		return result, nil
	case manifestv5.DockerV2Schema1MediaType,
		manifestv5.DockerV2Schema1SignedMediaType:
		// manifest is docker image schema1
		// return nil, fmt.Errorf("unsupported to scan for deprecated image MIME type %q",
		// 	s.mime)

		result, err := s.scanDockerV2Schema1MediaType(ctx, opts)
		if err != nil {
			return nil, err
		}
		return result, nil

	case imgspecv1.MediaTypeImageManifest:
		// manifest is oci image
		result, err := s.scanMediaTypeImageManifest(ctx, opts)
		if err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported MIME %q of image [%v]",
			s.mime, s.referenceName)
	}
}

func (s *Source) scanDockerV2ListMediaType(
	ctx context.Context,
	opts *ScanOptions,
) (*scan.Result, int, error) {
	var scannedNum int
	var errs []error
	var imageResults []*scan.ImageResult
	for _, m := range s.schema2List.Manifests {
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		osVersion := m.Platform.OSVersion
		osFeatures := m.Platform.OSFeatures
		variant := m.Platform.Variant
		dig := m.Digest
		// mime := m.MediaType

		// Skip image
		if !opts.Set.Allow(arch, osInfo, variant) {
			continue
		}
		refName := fmt.Sprintf("%s/%s/%s@%s", s.registry, s.project, s.name, dig)
		imageResult, err := scanImage(ctx, &scan.ScanOption{
			ReferenceName: refName,
			Digest:        dig,
			Platform: scan.Platform{
				Arch:       arch,
				OS:         osInfo,
				OSVersion:  osVersion,
				OSFeatures: osFeatures,
				Variant:    variant,
			},
		})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		scannedNum++
		imageResults = append(imageResults, imageResult)
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		return nil, scannedNum, fmt.Errorf(
			"error occurred when scan image [%v]: %w",
			s.referenceName, err,
		)
	}
	result := scan.NewResult(s.ReferenceNameWithoutTransport(), imageResults)
	return result, scannedNum, nil
}

func (s *Source) scanMediaTypeImageIndex(
	ctx context.Context,
	opts *ScanOptions,
) (*scan.Result, int, error) {
	var copiedNum int
	var errs []error
	var imageResults []*scan.ImageResult
	for _, m := range s.ociIndex.Manifests {
		// mime := m.MediaType
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		osVersion := m.Platform.OSVersion
		osFeatures := m.Platform.OSFeatures
		variant := m.Platform.Variant
		dig := m.Digest

		// skip image
		if !opts.Set.Allow(arch, osInfo, variant) {
			continue
		}
		// sourceRef, err := alltransportsv5.ParseImageName()
		refName := fmt.Sprintf("%s/%s/%s@%s", s.registry, s.project, s.name, dig)
		imageResult, err := scanImage(ctx, &scan.ScanOption{
			ReferenceName: refName,
			Digest:        dig,
			Platform: scan.Platform{
				Arch:       arch,
				OS:         osInfo,
				OSVersion:  osVersion,
				OSFeatures: osFeatures,
				Variant:    variant,
			},
		})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		copiedNum++
		imageResults = append(imageResults, imageResult)
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		return nil, copiedNum, fmt.Errorf(
			"error occurred when scan image [%v]: %w",
			s.referenceName, err,
		)
	}
	result := scan.NewResult(s.ReferenceNameWithoutTransport(), imageResults)
	return result, copiedNum, nil
}

func (s *Source) scanDockerV2Schema2MediaType(
	ctx context.Context,
	opts *ScanOptions,
) (*scan.Result, error) {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !opts.Set.Allow(arch, osInfo, variant) {
		return nil, utils.ErrNoAvailableImage
	}
	refName := s.ReferenceNameWithoutTransport()
	imageResult, err := scanImage(ctx, &scan.ScanOption{
		ReferenceName: refName,
		Digest:        s.digest,
		Platform: scan.Platform{
			Arch:    arch,
			OS:      osInfo,
			Variant: variant,
		},
	})
	if err != nil {
		return nil, err
	}
	result := scan.NewResult(refName, []*scan.ImageResult{
		imageResult,
	})

	return result, nil
}

func (s *Source) scanDockerV2Schema1MediaType(
	ctx context.Context,
	opts *ScanOptions,
) (*scan.Result, error) {
	arch := s.imageInspectInfo.Architecture
	osInfo := s.imageInspectInfo.Os
	variant := s.imageInspectInfo.Variant

	// skip image
	if !opts.Set.Allow(arch, osInfo, variant) {
		return nil, utils.ErrNoAvailableImage
	}

	refName := s.ReferenceNameWithoutTransport()
	imageResult, err := scanImage(ctx, &scan.ScanOption{
		ReferenceName: refName,
		Digest:        s.digest,
		Platform: scan.Platform{
			Arch:    arch,
			OS:      osInfo,
			Variant: variant,
		},
	})
	if err != nil {
		return nil, err
	}
	result := scan.NewResult(refName, []*scan.ImageResult{
		imageResult,
	})
	return result, nil
}

func (s *Source) scanMediaTypeImageManifest(
	ctx context.Context,
	opts *ScanOptions,
) (*scan.Result, error) {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !opts.Set.Allow(arch, osInfo, variant) {
		return nil, utils.ErrNoAvailableImage
	}

	refName := s.ReferenceNameWithoutTransport()
	imageResult, err := scanImage(ctx, &scan.ScanOption{
		ReferenceName: refName,
		Digest:        s.digest,
		Platform: scan.Platform{
			Arch:    arch,
			OS:      osInfo,
			Variant: variant,
		},
	})
	if err != nil {
		return nil, err
	}
	result := scan.NewResult(refName, []*scan.ImageResult{
		imageResult,
	})

	return result, nil
}

func scanImage(
	ctx context.Context,
	o *scan.ScanOption,
) (*scan.ImageResult, error) {
	return scan.Scan(ctx, o)
}
