package source

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cnrancher/hangar/pkg/image/sign"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"

	"github.com/containers/common/pkg/retry"
	copyv5 "github.com/containers/image/v5/copy"
	manifestv5 "github.com/containers/image/v5/manifest"
	signaturev5 "github.com/containers/image/v5/signature"
	alltransportsv5 "github.com/containers/image/v5/transports/alltransports"
	typesv5 "github.com/containers/image/v5/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type SignV1Options struct {
	SigstorePrivateKey string
	SigstorePassphrase []byte
	Set                types.FilterSet
	Policy             *signaturev5.Policy
}

func (s *Source) SignV1(
	ctx context.Context,
	opts *SignV1Options,
) error {
	switch s.mime {
	case manifestv5.DockerV2ListMediaType:
		// manifest is docker image list
		num, err := s.signV1DockerV2ListMediaType(ctx, opts)
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
		num, err := s.signV1MediaTypeImageIndex(ctx, opts)
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
		err := s.signV1DockerV2Schema2MediaType(ctx, opts)
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
		err := s.signV1MediaTypeImageManifest(ctx, opts)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported MIME %q of image [%v]",
			s.mime, s.referenceName)
	}
}

func (s *Source) signV1DockerV2ListMediaType(
	ctx context.Context,
	opts *SignV1Options,
) (int, error) {
	var signedNum int
	var errs []error
	for _, m := range s.schema2List.Manifests {
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		variant := m.Platform.Variant
		dig := m.Digest
		mime := m.MediaType

		// Skip image
		if !opts.Set.Allow(arch, osInfo, variant) {
			continue
		}
		sourceRef, err := alltransportsv5.ParseImageName(fmt.Sprintf("%s%s/%s/%s@%s",
			s.imageType.Transport(), s.registry, s.project, s.name, dig))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = signImageV1(ctx, &signV1Options{
			sigstorePrivateKey:           opts.SigstorePrivateKey,
			sigstorePrivateKeyPassphrase: opts.SigstorePassphrase,

			sourceRef:  sourceRef,
			sourceCtx:  s.systemCtx,
			policy:     opts.Policy,
			sourceMIME: mime,
		})
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

func (s *Source) signV1MediaTypeImageIndex(
	ctx context.Context,
	opts *SignV1Options,
) (int, error) {
	var copiedNum int
	var errs []error
	for _, m := range s.ociIndex.Manifests {
		mime := m.MediaType
		arch := m.Platform.Architecture
		osInfo := m.Platform.OS
		variant := m.Platform.Variant
		dig := m.Digest

		// skip image
		if !opts.Set.Allow(arch, osInfo, variant) {
			continue
		}
		sourceRef, err := alltransportsv5.ParseImageName(fmt.Sprintf("%s%s/%s/%s@%s",
			s.imageType.Transport(), s.registry, s.project, s.name, dig))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = signImageV1(ctx, &signV1Options{
			sigstorePrivateKey:           opts.SigstorePrivateKey,
			sigstorePrivateKeyPassphrase: opts.SigstorePassphrase,

			sourceRef:  sourceRef,
			sourceCtx:  s.systemCtx,
			policy:     opts.Policy,
			sourceMIME: mime,
		})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		copiedNum++
	}

	if len(errs) > 0 {
		err := errors.Join(errs...)
		return copiedNum, fmt.Errorf(
			"error occurred when sign image [%v]: %w",
			s.referenceName, err,
		)
	}
	return copiedNum, nil
}

func (s *Source) signV1DockerV2Schema2MediaType(
	ctx context.Context,
	opts *SignV1Options,
) error {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !opts.Set.Allow(arch, osInfo, variant) {
		return utils.ErrNoAvailableImage
	}
	sourceRef, err := s.Reference()
	if err != nil {
		return err
	}

	err = signImageV1(ctx, &signV1Options{
		sigstorePrivateKey:           opts.SigstorePrivateKey,
		sigstorePrivateKeyPassphrase: opts.SigstorePassphrase,

		sourceRef:  sourceRef,
		sourceCtx:  s.systemCtx,
		policy:     opts.Policy,
		sourceMIME: s.mime,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Source) signV1MediaTypeImageManifest(
	ctx context.Context,
	opts *SignV1Options,
) error {
	arch := s.ociConfig.Architecture
	osInfo := s.ociConfig.OS
	variant := s.ociConfig.Variant

	// Skip image
	if !opts.Set.Allow(arch, osInfo, variant) {
		return utils.ErrNoAvailableImage
	}

	sourceRef, err := s.Reference()
	if err != nil {
		return err
	}
	err = signImageV1(ctx, &signV1Options{
		sigstorePrivateKey:           opts.SigstorePrivateKey,
		sigstorePrivateKeyPassphrase: opts.SigstorePassphrase,

		sourceRef:  sourceRef,
		sourceCtx:  s.systemCtx,
		policy:     opts.Policy,
		sourceMIME: s.mime,
	})
	if err != nil {
		return err
	}

	return nil
}

type signV1Options struct {
	sigstorePrivateKey           string
	sigstorePrivateKeyPassphrase []byte
	sourceRef                    typesv5.ImageReference
	sourceCtx                    *typesv5.SystemContext
	policy                       *signaturev5.Policy
	sourceMIME                   string
}

func signImageV1(
	ctx context.Context,
	o *signV1Options,
) error {
	copyOpts := &copyv5.Options{
		SignBySigstorePrivateKeyFile:     o.sigstorePrivateKey,
		SignSigstorePrivateKeyPassphrase: o.sigstorePrivateKeyPassphrase,

		ReportWriter:         nil,
		SourceCtx:            utils.CopySystemContext(o.sourceCtx),
		DestinationCtx:       utils.CopySystemContext(o.sourceCtx),
		PreserveDigests:      true,
		MaxParallelDownloads: 3,
	}
	switch o.sourceMIME {
	case manifestv5.DockerV2Schema1MediaType,
		manifestv5.DockerV2Schema1SignedMediaType:
		return fmt.Errorf("signImage: MIME type %q is not supported to sign",
			o.sourceMIME)
	case manifestv5.DockerV2ListMediaType,
		imgspecv1.MediaTypeImageIndex:
		return fmt.Errorf("signImage: the image MIME type should be a single image, not %q",
			o.sourceMIME)
	}

	signer := sign.NewSigner(&sign.SignerOption{
		CopyOptions: copyOpts,
		RetryOptions: &retry.Options{
			MaxRetry: 3,
			Delay:    time.Microsecond * 100,
		},

		Reference: o.sourceRef,
		Policy:    o.policy,
	})
	logrus.Debugf("Start sign image %q with key %q",
		o.sourceRef.DockerReference(), copyOpts.SignBySigstorePrivateKeyFile)
	return signer.Sign(ctx)
}
