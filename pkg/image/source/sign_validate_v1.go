package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/cnrancher/hangar/pkg/image/sign"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/sirupsen/logrus"

	manifestv5 "github.com/containers/image/v5/manifest"
	typesv5 "github.com/containers/image/v5/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func (s *Source) ValidateSignatureV1(
	ctx context.Context,
	publicKey string,
	repository string,
	set types.FilterSet,
) error {
	switch s.mime {
	case manifestv5.DockerV2ListMediaType:
		// manifest is docker image list
		num, err := s.validateSignatureV1DockerV2ListMediaType(ctx, publicKey, repository, set)
		if err != nil {
			return err
		}
		logrus.Debugf("validated [%d] images", num)
		if num == 0 {
			return utils.ErrNoAvailableImage
		}
		return nil
	case imgspecv1.MediaTypeImageIndex:
		// manifest is oci image list
		num, err := s.validateSignatureV1MediaTypeImageIndex(ctx, publicKey, repository, set)
		if err != nil {
			return err
		}
		logrus.Debugf("validated [%d] images", num)
		if num == 0 {
			return utils.ErrNoAvailableImage
		}
		return nil
	case manifestv5.DockerV2Schema2MediaType:
		// manifest is docker image schema2
		err := s.validateSignatureV1DockerV2Schema2MediaType(ctx, publicKey, repository, set)
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
		err := s.validateSignatureV1MediaTypeImageManifest(ctx, publicKey, repository, set)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported MIME %q of image [%v]",
			s.mime, s.referenceName)
	}
}

func (s *Source) validateSignatureV1DockerV2ListMediaType(
	ctx context.Context,
	publicKey string,
	repository string,
	set types.FilterSet,
) (int, error) {
	var validateNum int
	var errs []error
	for _, m := range s.schema2List.Manifests {
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		variant := m.Platform.Variant
		dig := m.Digest

		// Skip image
		if !set.Allow(arch, osInfo, variant) {
			continue
		}
		sourceRef, err := alltransports.ParseImageName(fmt.Sprintf(
			"%s%s/%s/%s@%s",
			s.imageType.Transport(), s.registry, s.project, s.name, dig))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := validateImageSignatureV1(ctx, &signValidateV1Options{
			sigstorePubkey: publicKey,
			repository:     repository,
			imageRef:       sourceRef,
			sysCtx:         s.systemCtx,
		}); err != nil {
			errs = append(errs, err)
			continue
		}
		validateNum++
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		return validateNum, fmt.Errorf(
			"error occurred when validate image [%v]: %w",
			s.referenceName, err,
		)
	}
	return validateNum, nil
}

func (s *Source) validateSignatureV1MediaTypeImageIndex(
	ctx context.Context,
	publicKey string,
	repository string,
	set types.FilterSet,
) (int, error) {
	var validateNum int
	var errs []error
	for _, m := range s.ociIndex.Manifests {
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		variant := m.Platform.Variant
		dig := m.Digest

		// Skip image
		if !set.Allow(arch, osInfo, variant) {
			continue
		}
		sourceRef, err := alltransports.ParseImageName(fmt.Sprintf(
			"%s%s/%s/%s@%s",
			s.imageType.Transport(), s.registry, s.project, s.name, dig))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := validateImageSignatureV1(ctx, &signValidateV1Options{
			sigstorePubkey: publicKey,
			repository:     repository,
			imageRef:       sourceRef,
			sysCtx:         s.systemCtx,
		}); err != nil {
			errs = append(errs, err)
			continue
		}
		validateNum++
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		return validateNum, fmt.Errorf(
			"error occurred when validate image [%v]: %w",
			s.referenceName, err,
		)
	}
	return validateNum, nil
}

func (s *Source) validateSignatureV1DockerV2Schema2MediaType(
	ctx context.Context,
	publicKey string,
	repository string,
	set types.FilterSet,
) error {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !set.Allow(arch, osInfo, variant) {
		return utils.ErrNoAvailableImage
	}

	sourceRef, err := s.Reference()
	if err != nil {
		return err
	}

	if err := validateImageSignatureV1(ctx, &signValidateV1Options{
		sigstorePubkey: publicKey,
		repository:     repository,
		imageRef:       sourceRef,
		sysCtx:         s.systemCtx,
	}); err != nil {
		return err
	}
	return nil
}

func (s *Source) validateSignatureV1MediaTypeImageManifest(
	ctx context.Context,
	publicKey string,
	repository string,
	set types.FilterSet,
) error {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !set.Allow(arch, osInfo, variant) {
		return utils.ErrNoAvailableImage
	}

	sourceRef, err := s.Reference()
	if err != nil {
		return err
	}
	if err := validateImageSignatureV1(ctx, &signValidateV1Options{
		sigstorePubkey: publicKey,
		repository:     repository,
		imageRef:       sourceRef,
		sysCtx:         s.systemCtx,
	}); err != nil {
		return err
	}
	return nil
}

type signValidateV1Options struct {
	sigstorePubkey string
	repository     string
	imageRef       typesv5.ImageReference
	sysCtx         *typesv5.SystemContext
}

func validateImageSignatureV1(
	ctx context.Context,
	opts *signValidateV1Options,
) error {
	v := sign.NewValidator(&sign.ValidatorOption{
		Reference:     opts.imageRef,
		Repository:    opts.repository,
		Pubkey:        opts.sigstorePubkey,
		SystemContext: opts.sysCtx,
	})
	if err := v.Validate(ctx); err != nil {
		return err
	}
	return nil
}
