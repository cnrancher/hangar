package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/cnrancher/hangar/pkg/image/internal/private"
	signv2 "github.com/cnrancher/hangar/pkg/image/sign_v2"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"

	"github.com/containers/common/pkg/retry"
	manifestv5 "github.com/containers/image/v5/manifest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ValidateV2Options struct {
	signv2.ValidatorOption

	ValidateManifestIndex bool

	Set types.FilterSet
}

func (s *Source) ValidateSignatureV2(
	ctx context.Context,
	opts *ValidateV2Options,
) ([]*signv2.ImageResult, error) {
	switch s.mime {
	case manifestv5.DockerV2ListMediaType:
		// manifest is docker image list
		results, err := s.validateSignatureV2DockerV2ListMediaType(ctx, opts)
		if err != nil {
			return results, err
		}
		logrus.Debugf("validated [%d] images", len(results))
		if len(results) == 0 {
			return results, utils.ErrNoAvailableImage
		}
		return results, nil
	case imgspecv1.MediaTypeImageIndex:
		// manifest is oci image list
		results, err := s.validateSignatureV2MediaTypeImageIndex(ctx, opts)
		if err != nil {
			return results, err
		}
		logrus.Debugf("validated [%d] images", len(results))
		if len(results) == 0 {
			return results, utils.ErrNoAvailableImage
		}
		return results, nil
	case manifestv5.DockerV2Schema2MediaType:
		// manifest is docker image schema2
		results, err := s.validateSignatureV2DockerV2Schema2MediaType(ctx, opts)
		if err != nil {
			return results, err
		}
		return results, nil
	case manifestv5.DockerV2Schema1MediaType,
		manifestv5.DockerV2Schema1SignedMediaType:
		// docker image schema1 is not supported to sign
		return nil, fmt.Errorf("unsupported to sign for deprecated image MIME type %q",
			s.mime)
	case imgspecv1.MediaTypeImageManifest:
		// manifest is oci image
		results, err := s.validateSignatureV2MediaTypeImageManifest(ctx, opts)
		if err != nil {
			return results, err
		}
		return results, nil
	default:
		return nil, fmt.Errorf("unsupported MIME %q of image [%v]",
			s.mime, s.referenceName)
	}
}

func (s *Source) validateSignatureV2DockerV2ListMediaType(
	ctx context.Context,
	opts *ValidateV2Options,
) ([]*signv2.ImageResult, error) {
	var errs []error
	var results = []*signv2.ImageResult{}

	if opts.ValidateManifestIndex {
		// Validate manifest index signature
		image := fmt.Sprintf("%v/%v/%v@%v", s.registry, s.project, s.name, s.manifestDigest)
		o := opts.ValidatorOption
		o.MediaType = s.mime
		o.Digest = s.manifestDigest
		r, err := validateImageSignatureV2(ctx, image, &o)
		if err != nil {
			errs = append(errs, err)
		} else {
			results = append(results, r)
		}
	}

	for _, m := range s.schema2List.Manifests {
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		variant := m.Platform.Variant
		dig := m.Digest

		// Skip image
		if !opts.Set.Allow(arch, osInfo, variant) {
			continue
		}
		image := fmt.Sprintf("%v/%v/%v@%v", s.registry, s.project, s.name, dig)
		o := opts.ValidatorOption
		o.MediaType = m.MediaType
		o.Digest = dig
		o.Platform.Arch = arch
		o.Platform.OS = osInfo
		o.Platform.Variant = variant
		o.Platform.OSFeatures = m.Platform.Features
		o.Platform.OSVersion = m.Platform.OSVersion
		r, err := validateImageSignatureV2(ctx, image, &o)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		results = append(results, r)
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		return results, fmt.Errorf(
			"error occurred when validate image [%v]: %w",
			s.referenceName, err,
		)
	}
	return results, nil
}

func (s *Source) validateSignatureV2MediaTypeImageIndex(
	ctx context.Context,
	opts *ValidateV2Options,
) ([]*signv2.ImageResult, error) {
	var errs []error
	var results = []*signv2.ImageResult{}

	if opts.ValidateManifestIndex {
		// Validate manifest index signature
		image := fmt.Sprintf("%v/%v/%v@%v", s.registry, s.project, s.name, s.manifestDigest)
		o := opts.ValidatorOption
		o.MediaType = s.mime
		o.Digest = s.manifestDigest
		r, err := validateImageSignatureV2(ctx, image, &o)
		if err != nil {
			errs = append(errs, err)
		} else {
			results = append(results, r)
		}
	}

	for _, m := range s.ociIndex.Manifests {
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		variant := m.Platform.Variant
		dig := m.Digest

		// Skip image
		if !opts.Set.Allow(arch, osInfo, variant) {
			continue
		}
		image := fmt.Sprintf("%v/%v/%v@%v", s.registry, s.project, s.name, dig)
		o := opts.ValidatorOption
		o.Digest = dig
		o.Platform.Arch = arch
		o.Platform.OS = osInfo
		o.MediaType = m.MediaType
		o.Platform.Variant = variant
		o.Platform.OSFeatures = m.Platform.OSFeatures
		o.Platform.OSVersion = m.Platform.OSVersion
		r, err := validateImageSignatureV2(ctx, image, &o)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		results = append(results, r)
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		return results, fmt.Errorf(
			"error occurred when validate image [%v]: %w",
			s.referenceName, err,
		)
	}
	return results, nil
}

func (s *Source) validateSignatureV2DockerV2Schema2MediaType(
	ctx context.Context,
	opts *ValidateV2Options,
) ([]*signv2.ImageResult, error) {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !opts.Set.Allow(arch, osInfo, variant) {
		return nil, utils.ErrNoAvailableImage
	}

	image := fmt.Sprintf("%v/%v/%v@%v", s.registry, s.project, s.name, s.manifestDigest)
	o := opts.ValidatorOption
	o.Digest = s.manifestDigest
	o.Platform.Arch = arch
	o.Platform.OS = osInfo
	o.MediaType = s.mime
	o.Platform.Variant = variant
	o.Platform.OSFeatures = s.ociConfig.OSFeatures
	o.Platform.OSVersion = s.ociConfig.OSVersion
	result, err := validateImageSignatureV2(ctx, image, &o)
	if err != nil {
		return nil, err
	}
	return []*signv2.ImageResult{result}, nil
}

func (s *Source) validateSignatureV2MediaTypeImageManifest(
	ctx context.Context,
	opts *ValidateV2Options,
) ([]*signv2.ImageResult, error) {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !opts.Set.Allow(arch, osInfo, variant) {
		return nil, utils.ErrNoAvailableImage
	}

	image := fmt.Sprintf("%v/%v/%v@%v", s.registry, s.project, s.name, s.manifestDigest)
	o := opts.ValidatorOption
	o.Digest = s.manifestDigest
	o.Platform.Arch = arch
	o.MediaType = s.mime
	o.Platform.OS = osInfo
	o.Platform.Variant = variant
	o.Platform.OSFeatures = s.ociConfig.OSFeatures
	o.Platform.OSVersion = s.ociConfig.OSVersion
	result, err := validateImageSignatureV2(ctx, image, &o)
	if err != nil {
		return nil, err
	}
	return []*signv2.ImageResult{result}, nil
}

func validateImageSignatureV2(
	ctx context.Context,
	image string,
	opts *signv2.ValidatorOption,
) (*signv2.ImageResult, error) {
	v := signv2.NewValidator(opts, image)
	logrus.Debugf("Start validate image [%v] with key [%v] OIDC Provider [%v]",
		image, opts.KeyRef, opts.CertOidcIssuer)

	err := retry.IfNecessary(ctx, func() error {
		return v.Validate(ctx)
	}, private.RetryOptions())
	if err != nil {
		return nil, err
	}
	return v.Result(), nil
}
