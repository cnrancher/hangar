package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/cnrancher/hangar/pkg/image/internal/private"
	signv2 "github.com/cnrancher/hangar/pkg/image/sign_v2"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/common/pkg/retry"
	manifestv5 "github.com/containers/image/v5/manifest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

type SignV2Options struct {
	signv2.SignerOption

	// Sign DockerV2ListMediaType and oci MediaTypeImageIndex
	SignManifestIndex bool

	Set types.FilterSet
}

func (s *Source) SignV2(
	ctx context.Context,
	opts *SignV2Options,
) error {
	switch s.mime {
	case manifestv5.DockerV2ListMediaType:
		// manifest is docker image list
		num, err := s.signV2DockerV2ListMediaType(ctx, opts)
		if err != nil {
			return err
		}
		logrus.Debugf("signed [%d] images", num)
		if num == 0 {
			return utils.ErrNoAvailableImage
		}
		return nil
	case imgspecv1.MediaTypeImageIndex:
		// manifest is oci image list
		num, err := s.signV2MediaTypeImageIndex(ctx, opts)
		if err != nil {
			return err
		}
		logrus.Debugf("signed [%d] images", num)
		if num == 0 {
			return utils.ErrNoAvailableImage
		}
		return nil
	case manifestv5.DockerV2Schema2MediaType:
		// manifest is docker image schema2
		err := s.signV2DockerV2Schema2MediaType(ctx, opts)
		if err != nil {
			return err
		}
		return nil
	case manifestv5.DockerV2Schema1MediaType,
		manifestv5.DockerV2Schema1SignedMediaType:
		// docker image schema1 is not supported to sign
		return fmt.Errorf("unsupported to sign for deprecated image MIME type %q",
			s.mime)
	case imgspecv1.MediaTypeImageManifest:
		// manifest is oci image
		err := s.signV2MediaTypeImageManifest(ctx, opts)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported MIME %q of image [%v]",
			s.mime, s.referenceName)
	}
}

func (s *Source) signV2DockerV2ListMediaType(
	ctx context.Context,
	opts *SignV2Options,
) (int, error) {
	var signedNum int
	var errs []error
	if opts.SignManifestIndex {
		image := fmt.Sprintf("%s/%s/%s@%s",
			s.registry, s.project, s.name, s.manifestDigest)

		err := signImageV2(ctx, &opts.SignerOption, image)
		if err != nil {
			errs = append(errs, err)
		} else {
			signedNum++
		}
	}

	for _, m := range s.schema2List.Manifests {
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		variant := m.Platform.Variant
		dig := m.Digest
		// mime := m.MediaType

		// Skip image
		if !opts.Set.Allow(arch, osInfo, variant) {
			continue
		}
		image := fmt.Sprintf("%s/%s/%s@%s",
			s.registry, s.project, s.name, dig)

		err := signImageV2(ctx, &opts.SignerOption, image)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		signedNum++
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		return signedNum, fmt.Errorf(
			"error occurred when sign image [%v]: %w",
			s.referenceName, err,
		)
	}
	return signedNum, nil
}

func (s *Source) signV2MediaTypeImageIndex(
	ctx context.Context,
	opts *SignV2Options,
) (int, error) {
	var signedNum int
	var errs []error

	if opts.SignManifestIndex {
		image := fmt.Sprintf("%s/%s/%s@%s",
			s.registry, s.project, s.name, s.manifestDigest)

		err := signImageV2(ctx, &opts.SignerOption, image)
		if err != nil {
			errs = append(errs, err)
		} else {
			signedNum++
		}
	}

	for _, m := range s.ociIndex.Manifests {
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		variant := m.Platform.Variant
		dig := m.Digest

		// skip image
		if !opts.Set.Allow(arch, osInfo, variant) {
			continue
		}
		image := fmt.Sprintf("%s/%s/%s@%s",
			s.registry, s.project, s.name, dig)
		err := signImageV2(ctx, &opts.SignerOption, image)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		signedNum++
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		return signedNum, fmt.Errorf(
			"error occurred when sign image [%v]: %w",
			s.referenceName, err,
		)
	}
	return signedNum, nil
}

func (s *Source) signV2DockerV2Schema2MediaType(
	ctx context.Context,
	opts *SignV2Options,
) error {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !opts.Set.Allow(arch, osInfo, variant) {
		return utils.ErrNoAvailableImage
	}
	image := fmt.Sprintf("%s/%s/%s@%s",
		s.registry, s.project, s.name, s.manifestDigest)
	err := signImageV2(ctx, &opts.SignerOption, image)
	if err != nil {
		return err
	}

	return nil
}

func (s *Source) signV2MediaTypeImageManifest(
	ctx context.Context,
	opts *SignV2Options,
) error {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !opts.Set.Allow(arch, osInfo, variant) {
		return utils.ErrNoAvailableImage
	}
	image := fmt.Sprintf("%s/%s/%s@%s",
		s.registry, s.project, s.name, s.manifestDigest)
	err := signImageV2(ctx, &opts.SignerOption, image)
	if err != nil {
		return err
	}

	return nil
}

func signImageV2(
	ctx context.Context,
	o *signv2.SignerOption,
	image string,
) error {
	signer := signv2.NewSigner(o, image)
	logrus.Debugf("Start sign image [%v] with key [%v] OIDC Provider [%v]",
		image, o.Key, o.OIDCProvider)

	err := retry.IfNecessary(ctx, func() error {
		return signer.Sign(ctx)
	}, private.RetryOptions())

	return err
}
