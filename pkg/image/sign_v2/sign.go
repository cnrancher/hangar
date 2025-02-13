// https://github.com/sigstore/cosign/blob/v2.4.2/cmd/cosign/cli/sign/sign.go

package signv2

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/rekor"
	cremote "github.com/sigstore/cosign/v2/pkg/cosign/remote"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/walk"
	sigPayload "github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/sirupsen/logrus"

	cosign_internal "github.com/cnrancher/hangar/pkg/image/sign_v2/internal/cosign"
	payload_internal "github.com/cnrancher/hangar/pkg/image/sign_v2/internal/payload"
	rekor_internal "github.com/cnrancher/hangar/pkg/image/sign_v2/internal/rekor"
)

var (
	globalSV      *SignerVerifier
	globalSVMutex = &sync.Mutex{}
)

type Signer struct {
	image                   string
	key                     string
	recursive               bool
	tlogUpload              bool
	recordCreationTimestamp bool
	rekorURL                string
	oidcIssuer              string
	oidcClientID            string
	oidcProvider            string
	insecureSkipTLSVerify   bool
	skipConfotmation        bool
	authConfig              authn.AuthConfig

	signer cosign_internal.Signer
}

type SignerOption struct {
	Key                     string
	Recursive               bool
	TlogUpload              bool
	RecordCreationTimestamp bool
	RekorURL                string
	OIDCIssuer              string
	OIDCClientID            string
	OIDCProvider            string

	InsecureSkipTLSVerify bool
	SkipConfotmation      bool
	AuthConfig            authn.AuthConfig
}

func NewSigner(o *SignerOption, image string) *Signer {
	s := &Signer{
		image:                   image,
		key:                     o.Key,
		recursive:               o.Recursive,
		tlogUpload:              o.TlogUpload,
		recordCreationTimestamp: o.RecordCreationTimestamp,
		rekorURL:                o.RekorURL,
		oidcIssuer:              o.OIDCIssuer,
		oidcClientID:            o.OIDCClientID,
		oidcProvider:            o.OIDCProvider,
		insecureSkipTLSVerify:   o.InsecureSkipTLSVerify,
		skipConfotmation:        o.SkipConfotmation,
		authConfig:              o.AuthConfig,
		signer:                  nil,
	}
	return s
}

func InitGlobalSignerVerifier(
	ctx context.Context,
	key string,
	ko *options.KeyOpts,
) error {
	globalSVMutex.Lock()
	defer globalSVMutex.Unlock()

	if globalSV != nil {
		return nil
	}

	var err error
	genKey := false
	switch {
	case key != "":
		globalSV, err = signerFromKeyRef(ctx, key, ko.PassFunc)
	default:
		genKey = true
		logrus.Infof("Generating ephemeral keys...")
		globalSV, err = signerFromNewKey()
	}
	if err != nil {
		return err
	}
	if genKey {
		if globalSV, err = keylessSigner(ctx, ko, globalSV); err != nil {
			return err
		}
	}
	return nil
}

// Sign method is based on the `SignCmd` method of the cosign cli.
// Reference: https://github.com/sigstore/cosign/blob/v2.4.2/cmd/cosign/cli/sign/sign.go#L133
func (s *Signer) Sign(ctx context.Context) error {
	logrus.Debugf("Start sign image %v", s.image)
	dd := cremote.NewDupeDetector(globalSV)
	// Set up an ErrDone consideration to return along "success" paths
	var ErrDone error
	if !s.recursive {
		ErrDone = mutate.ErrSkipChildren
	}

	opts := s.ClientOpts(ctx)
	ref, err := name.ParseReference(s.image, s.NameOptions()...)
	if err != nil {
		return fmt.Errorf("failed to parse reference: %w", err)
	}
	if digest, ok := ref.(name.Digest); ok && !s.recursive {
		se, err := ociremote.SignedEntity(ref, opts...)
		if _, isEntityNotFoundErr := err.(*ociremote.EntityNotFoundError); isEntityNotFoundErr {
			se = ociremote.SignedUnknown(digest)
		} else if err != nil {
			return fmt.Errorf("accessing image: %w", err)
		}
		err = s.signDigest(ctx, digest, dd, se)
		if err != nil {
			return fmt.Errorf("signing digest: %w", err)
		}
		return nil
	}

	se, err := ociremote.SignedEntity(ref, opts...)
	if err != nil {
		return fmt.Errorf("accessing entity: %w", err)
	}
	if err := walk.SignedEntity(ctx, se, func(ctx context.Context, se oci.SignedEntity) error {
		// Get the digest for this entity in our walk.payload
		d, err := se.(interface{ Digest() (v1.Hash, error) }).Digest()
		if err != nil {
			return fmt.Errorf("computing digest: %w", err)
		}
		digest := ref.Context().Digest(d.String())
		err = s.signDigest(ctx, digest, dd, se)
		if err != nil {
			return fmt.Errorf("signing digest: %w", err)
		}
		return ErrDone
	}); err != nil {
		return fmt.Errorf("recursively signing: %w", err)
	}
	return nil
}

func (s *Signer) signDigest(
	ctx context.Context,
	digest name.Digest,
	dd mutate.DupeDetector,
	se oci.SignedEntity,
) error {
	// The payload can be passed to skip generation.
	payload, err := (&sigPayload.Cosign{
		Image:           digest,
		ClaimedIdentity: "",
		Annotations:     nil,
	}).MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to generate payload: %w", err)
	}

	s.signer = payload_internal.NewSigner(globalSV)
	if s.tlogUpload {
		rClient, err := rekor.NewClient(s.rekorURL)
		if err != nil {
			return err
		}
		s.signer = rekor_internal.WrapSigner(s.signer, rClient)
	}

	ociSig, _, err := s.signer.Sign(ctx, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	// Attach the signature to the entity.
	newSE, err := mutate.AttachSignatureToEntity(
		se, ociSig, mutate.WithDupeDetector(dd),
		mutate.WithRecordCreationTimestamp(false))
	if err != nil {
		return err
	}
	// Publish the signatures associated with this entity
	walkOpts := s.ClientOpts(ctx)
	// Check if we are overriding the signatures repository location
	repo, _ := ociremote.GetEnvTargetRepository()
	if repo.RepositoryStr() == "" {
		logrus.Debugf("Pushing signature to: %s", digest.Repository)
	} else {
		logrus.Debugf("Pushing signature to: %s", repo.RepositoryStr())
	}

	// Publish the signatures associated with this entity
	return ociremote.WriteSignatures(digest.Repository, newSE, walkOpts...)
}
