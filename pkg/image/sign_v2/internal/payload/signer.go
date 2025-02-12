// https://github.com/sigstore/cosign/tree/main/internal/pkg/cosign

package payload

import (
	"bytes"
	"context"
	"crypto"
	"encoding/base64"
	"io"

	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/sigstore/pkg/signature"
	signatureoptions "github.com/sigstore/sigstore/pkg/signature/options"

	"github.com/cnrancher/hangar/pkg/image/sign_v2/internal/cosign"
)

type payloadSigner struct {
	payloadSigner signature.Signer
}

// Sign implements `cosign_internal.Signer`
func (ps *payloadSigner) Sign(ctx context.Context, payload io.Reader) (oci.Signature, crypto.PublicKey, error) {
	payloadBytes, err := io.ReadAll(payload)
	if err != nil {
		return nil, nil, err
	}
	sig, err := ps.signPayload(ctx, payloadBytes)
	if err != nil {
		return nil, nil, err
	}
	pk, err := ps.publicKey(ctx)
	if err != nil {
		return nil, nil, err
	}

	b64sig := base64.StdEncoding.EncodeToString(sig)
	ociSig, err := static.NewSignature(payloadBytes, b64sig)
	if err != nil {
		return nil, nil, err
	}
	return ociSig, pk, nil
}

func (ps *payloadSigner) publicKey(ctx context.Context) (pk crypto.PublicKey, err error) {
	pkOpts := []signature.PublicKeyOption{signatureoptions.WithContext(ctx)}
	pk, err = ps.payloadSigner.PublicKey(pkOpts...)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

func (ps *payloadSigner) signPayload(ctx context.Context, payloadBytes []byte) (sig []byte, err error) {
	sOpts := []signature.SignOption{signatureoptions.WithContext(ctx)}
	sig, err = ps.payloadSigner.SignMessage(bytes.NewReader(payloadBytes), sOpts...)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func NewSigner(s signature.Signer) cosign.Signer {
	return &payloadSigner{
		payloadSigner: s,
	}
}
