// https://github.com/sigstore/cosign/tree/main/internal/pkg/cosign

package cosign

import (
	"context"
	"crypto"
	"io"

	"github.com/sigstore/cosign/v2/pkg/oci"
)

// Signer signs payloads in the form of `oci.Signature`s
type Signer interface {
	// Sign signs the given payload, returning the results as an `oci.Signature` which can be verified using the returned `crypto.PublicKey`.
	Sign(ctx context.Context, payload io.Reader) (oci.Signature, crypto.PublicKey, error)
}
