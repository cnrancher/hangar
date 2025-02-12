// https://github.com/sigstore/cosign/tree/main/internal/pkg/cosign

package fulcio

import (
	"context"
	"crypto"
	"io"

	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"

	"github.com/cnrancher/hangar/pkg/image/sign_v2/internal/cosign"
)

// signerWrapper still needs to actually upload keys to Fulcio and receive
// the resulting `Cert` and `Chain`, which are added to the returned `oci.Signature`
type signerWrapper struct {
	inner cosign.Signer

	cert, chain []byte
}

var _ cosign.Signer = (*signerWrapper)(nil)

// Sign implements `cosign.Signer`
func (fs *signerWrapper) Sign(ctx context.Context, payload io.Reader) (oci.Signature, crypto.PublicKey, error) {
	sig, pub, err := fs.inner.Sign(ctx, payload)
	if err != nil {
		return nil, nil, err
	}

	newSig, err := mutate.Signature(sig, mutate.WithCertChain(fs.cert, fs.chain))
	if err != nil {
		return nil, nil, err
	}

	return newSig, pub, nil
}

// WrapSigner returns a `cosign.Signer` which leverages Fulcio to create a Cert and Chain for the signature
func WrapSigner(inner cosign.Signer, cert, chain []byte) cosign.Signer {
	return &signerWrapper{
		inner: inner,
		cert:  cert,
		chain: chain,
	}
}
