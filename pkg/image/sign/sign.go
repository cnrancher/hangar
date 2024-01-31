package sign

import (
	"context"
	"fmt"

	"github.com/containers/common/pkg/retry"
	copyv5 "github.com/containers/image/v5/copy"
	signaturev5 "github.com/containers/image/v5/signature"
	typesv5 "github.com/containers/image/v5/types"
)

type Signer struct {
	reference    typesv5.ImageReference
	retryOptions *retry.Options
	policy       *signaturev5.Policy
	copyOptions  *copyv5.Options
}

type SignerOption struct {
	Reference    typesv5.ImageReference
	RetryOptions *retry.Options
	Policy       *signaturev5.Policy

	CopyOptions *copyv5.Options
}

func NewSigner(o *SignerOption) *Signer {
	s := &Signer{
		reference:    o.Reference,
		retryOptions: o.RetryOptions,
		policy:       o.Policy,
		copyOptions:  o.CopyOptions,
	}
	return s
}

func (s *Signer) Sign(ctx context.Context) error {
	policyContext, err := signaturev5.NewPolicyContext(s.policy)
	if err != nil {
		return fmt.Errorf("copy: failed to create policy context: %w", err)
	}

	// The `containers/signature` does not provide a API to stand-alone sign an
	// image by sigstore private key file, so we need to use the
	// `containers/copy` library to sign the image by copying itself.
	err = retry.IfNecessary(ctx, func() error {
		var err error
		_, err = copyv5.Image(
			ctx,
			policyContext,
			s.reference,
			s.reference,
			s.copyOptions,
		)
		if err != nil {
			return err
		}
		return nil
	}, s.retryOptions)
	return err
}
