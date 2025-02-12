// https://github.com/sigstore/cosign/blob/v2.4.2/cmd/cosign/cli/sign/sign.go

package signv2

import (
	"context"
	"crypto"
	"fmt"

	"github.com/sigstore/cosign/v2/cmd/cosign/cli/fulcio"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/fulcio/fulcioverifier"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/cosign/pivkey"
	sigs "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sirupsen/logrus"
)

type SignerVerifier struct {
	Cert  []byte
	Chain []byte
	signature.SignerVerifier
	close func()
}

func signerFromSecurityKey(keySlot string) (*SignerVerifier, error) {
	sk, err := pivkey.GetKeyWithSlot(keySlot)
	if err != nil {
		return nil, err
	}
	sv, err := sk.SignerVerifier()
	if err != nil {
		sk.Close()
		return nil, err
	}

	// Handle the -cert flag.
	// With PIV, we assume the certificate is in the same slot on the PIV
	// token as the private key. If it's not there, show a warning to the
	// user.
	certFromPIV, err := sk.Certificate()
	var pemBytes []byte
	if err != nil {
		logrus.Warnf("no x509 certificate retrieved from the PIV token")
	} else {
		pemBytes, err = cryptoutils.MarshalCertificateToPEM(certFromPIV)
		if err != nil {
			sk.Close()
			return nil, err
		}
	}

	return &SignerVerifier{
		Cert:           pemBytes,
		SignerVerifier: sv,
		close:          sk.Close,
	}, nil
}

func signerFromKeyRef(
	ctx context.Context,
	keyRef string,
	passFunc cosign.PassFunc,
) (*SignerVerifier, error) {
	sv, err := sigs.SignerVerifierFromKeyRef(ctx, keyRef, passFunc)
	if err != nil {
		return nil, fmt.Errorf("reading key: %w", err)
	}
	certSigner := &SignerVerifier{
		SignerVerifier: sv,
	}

	return certSigner, nil
}

func signerFromNewKey() (*SignerVerifier, error) {
	privKey, err := cosign.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generating cert: %w", err)
	}
	sv, err := signature.LoadECDSASignerVerifier(privKey, crypto.SHA256)
	if err != nil {
		return nil, err
	}

	return &SignerVerifier{
		SignerVerifier: sv,
	}, nil
}

func keylessSigner(
	ctx context.Context,
	ko *options.KeyOpts,
	sv *SignerVerifier,
) (*SignerVerifier, error) {
	var (
		k   *fulcio.Signer
		err error
	)
	if ko.InsecureSkipFulcioVerify {
		if k, err = fulcio.NewSigner(ctx, *ko, sv); err != nil {
			return nil, fmt.Errorf("getting key from Fulcio: %w", err)
		}
	} else {
		if k, err = fulcioverifier.NewSigner(ctx, *ko, sv); err != nil {
			return nil, fmt.Errorf("getting key from Fulcio: %w", err)
		}
	}
	return &SignerVerifier{
		Cert:           k.Cert,
		Chain:          k.Chain,
		SignerVerifier: k,
	}, nil
}
