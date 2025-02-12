package sign

import (
	"context"
	"fmt"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"

	"github.com/containers/common/pkg/retry"
	imagev5 "github.com/containers/image/v5/image"
	signaturev5 "github.com/containers/image/v5/signature"
	typesv5 "github.com/containers/image/v5/types"
)

type Validator struct {
	reference     typesv5.ImageReference
	repository    string
	pubkey        string
	systemContext *typesv5.SystemContext
	retryOptions  *retry.Options
}

type ValidatorOption struct {
	// Reference is the image reference to be validate.
	Reference typesv5.ImageReference
	// If Repository is not empty, use the containers/image 'exactRepository'
	// signedIdentity to validate the signed image.
	// Validator will use the 'matchRepoDigestOrExact'
	Repository string
	// Pubkey is the sigstore public key file.
	Pubkey string
	// SystemContext
	SystemContext *typesv5.SystemContext
	// RetryOptions, can be nil
	RetryOptions *retry.Options
}

func NewValidator(o *ValidatorOption) *Validator {
	v := &Validator{
		reference:     o.Reference,
		repository:    o.Repository,
		pubkey:        o.Pubkey,
		retryOptions:  o.RetryOptions,
		systemContext: utils.CopySystemContext(o.SystemContext),
	}
	return v
}

func (v *Validator) Validate(ctx context.Context) error {
	policy, err := policyWithSigstorePubkey(
		v.reference.Transport().Name(), v.repository, v.pubkey)
	if err != nil {
		return fmt.Errorf("sign validate: failed to build policy: %w", err)
	}
	policyContext, err := signaturev5.NewPolicyContext(policy)
	if err != nil {
		return fmt.Errorf("sign validate: failed to create policy context: %w", err)
	}

	source, err := v.reference.NewImageSource(ctx, v.systemContext)
	if err != nil {
		return fmt.Errorf("sign validate: failed to create image source: %w", err)
	}
	img := imagev5.UnparsedInstance(source, nil)
	_, err = policyContext.IsRunningImageAllowed(ctx, img)
	if err != nil {
		return fmt.Errorf("sign validate: %w", err)
	}
	return nil
}

func policyWithSigstorePubkey(transport, repository, pubkey string) (*signaturev5.Policy, error) {
	var (
		identity signaturev5.PolicyReferenceMatch
		err      error
	)
	if repository != "" {
		identity, err = signaturev5.NewPRMExactRepository(repository)
		if err != nil {
			return nil, fmt.Errorf("signature NewPRMExactRepository failed: %w", err)
		}
		logrus.Debugf("Generate exactRepository signIdentity %v", utils.ToJSON(identity))
	} else {
		identity = signaturev5.NewPRMMatchRepoDigestOrExact()
		logrus.Debugf("Generate matchRepoDigestOrExact signIdentity %v",
			utils.ToJSON(identity))
	}

	requirement, err := signaturev5.NewPRSigstoreSignedKeyPath(pubkey, identity)
	// requirement, err := signaturev5.NewPRSigstoreSignedKeyPath(pubkey, nil)
	if err != nil {
		return nil, fmt.Errorf("signature NewPRSigstoreSignedKeyPath failed: %w", err)
	}
	policy := &signaturev5.Policy{
		Default: []signaturev5.PolicyRequirement{
			requirement,
		},
		Transports: map[string]signaturev5.PolicyTransportScopes{
			transport: {
				"": signaturev5.PolicyRequirements{
					requirement,
				},
			},
		},
	}
	return policy, nil
}
