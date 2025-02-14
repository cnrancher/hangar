package signv2

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/opencontainers/go-digest"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/fulcio"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/rekor"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/sign"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/verify"
	"github.com/sigstore/cosign/v2/pkg/blob"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/cosign/pivkey"
	"github.com/sigstore/cosign/v2/pkg/cosign/pkcs11key"
	sigs "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
)

type Validator struct {
	image  string
	result *ImageResult
	cmd    verify.VerifyCommand
}

type ValidatorOption struct {
	Digest    digest.Digest
	Platform  Platform
	MediaType string

	verify.VerifyCommand
}

func NewValidator(o *ValidatorOption, image string) *Validator {
	v := &Validator{
		image: image,
		result: &ImageResult{
			Digest:    o.Digest,
			Platform:  o.Platform,
			MediaType: o.MediaType,
		},
		cmd: o.VerifyCommand,
	}
	if v.cmd.HashAlgorithm == 0 {
		v.cmd.HashAlgorithm = crypto.SHA256
	}
	return v
}

func (v *Validator) Validate(ctx context.Context) error {
	identities := []cosign.Identity{}
	if v.cmd.KeyRef == "" {
		i, err := v.cmd.Identities()
		if err != nil {
			return err
		}
		identities = append(identities, i...)
	}

	ociremoteOpts, err := v.cmd.ClientOpts(ctx)
	if err != nil {
		return fmt.Errorf("failed to constructing client options: %w", err)
	}
	co := &cosign.CheckOpts{
		Annotations:                  v.cmd.Annotations.Annotations,
		RegistryClientOpts:           ociremoteOpts,
		CertGithubWorkflowTrigger:    v.cmd.CertGithubWorkflowTrigger,
		CertGithubWorkflowSha:        v.cmd.CertGithubWorkflowSha,
		CertGithubWorkflowName:       v.cmd.CertGithubWorkflowName,
		CertGithubWorkflowRepository: v.cmd.CertGithubWorkflowRepository,
		CertGithubWorkflowRef:        v.cmd.CertGithubWorkflowRef,
		IgnoreSCT:                    v.cmd.IgnoreSCT,
		SignatureRef:                 v.cmd.SignatureRef,
		PayloadRef:                   v.cmd.PayloadRef,
		Identities:                   identities,
		Offline:                      v.cmd.Offline,
		IgnoreTlog:                   v.cmd.IgnoreTlog,
		MaxWorkers:                   v.cmd.MaxWorkers,
		ExperimentalOCI11:            v.cmd.ExperimentalOCI11,
	}

	if v.cmd.CheckClaims {
		co.ClaimVerifier = cosign.SimpleClaimVerifier
	}

	if !v.cmd.IgnoreTlog {
		if v.cmd.RekorURL != "" {
			rekorClient, err := rekor.NewClient(v.cmd.RekorURL)
			if err != nil {
				return fmt.Errorf("creating Rekor client: %w", err)
			}
			co.RekorClient = rekorClient
		}
		// This performs an online fetch of the Rekor public keys, but this is needed
		// for verifying tlog entries (both online and offline).
		co.RekorPubKeys, err = cosign.GetRekorPubs(ctx)
		if err != nil {
			return fmt.Errorf("getting Rekor public keys: %w", err)
		}
	}

	if keylessVerification(v.cmd.KeyRef, v.cmd.Sk) {
		if v.cmd.CertChain != "" {
			chain, err := loadCertChainFromFileOrURL(v.cmd.CertChain)
			if err != nil {
				return err
			}

			co.RootCerts = x509.NewCertPool()
			co.RootCerts.AddCert(chain[len(chain)-1])
			if len(chain) > 1 {
				co.IntermediateCerts = x509.NewCertPool()
				for _, cert := range chain[:len(chain)-1] {
					co.IntermediateCerts.AddCert(cert)
				}
			}
		} else {
			// This performs an online fetch of the Fulcio roots. This is needed
			// for verifying keyless certificates (both online and offline).
			co.RootCerts, err = fulcio.GetRoots()
			if err != nil {
				return fmt.Errorf("getting Fulcio roots: %w", err)
			}
			co.IntermediateCerts, err = fulcio.GetIntermediates()
			if err != nil {
				return fmt.Errorf("getting Fulcio intermediates: %w", err)
			}
		}
	}

	// Ignore Signed Certificate Timestamp if the flag is set or a key is provided
	if shouldVerifySCT(v.cmd.IgnoreSCT, v.cmd.KeyRef, v.cmd.Sk) {
		co.CTLogPubKeys, err = cosign.GetCTLogPubs(ctx)
		if err != nil {
			return fmt.Errorf("getting ctlog public keys: %w", err)
		}
	}

	// Keys are optional!
	var pubKey signature.Verifier
	switch {
	case v.cmd.KeyRef != "":
		pubKey, err = sigs.PublicKeyFromKeyRefWithHashAlgo(ctx, v.cmd.KeyRef, v.cmd.HashAlgorithm)
		if err != nil {
			return fmt.Errorf("loading public key: %w", err)
		}

		pkcs11Key, ok := pubKey.(*pkcs11key.Key)
		if ok {
			defer pkcs11Key.Close()
		}
	case v.cmd.Sk:
		sk, err := pivkey.GetKeyWithSlot(v.cmd.Slot)
		if err != nil {
			return fmt.Errorf("opening piv token: %w", err)
		}
		defer sk.Close()
		pubKey, err = sk.Verifier()
		if err != nil {
			return fmt.Errorf("initializing piv token verifier: %w", err)
		}
	case v.cmd.CertRef != "":
		cert, err := loadCertFromFileOrURL(v.cmd.CertRef)
		if err != nil {
			return err
		}
		if v.cmd.CertChain == "" {
			// If no certChain is passed, the Fulcio root certificate will be used
			co.RootCerts, err = fulcio.GetRoots()
			if err != nil {
				return fmt.Errorf("getting Fulcio roots: %w", err)
			}
			co.IntermediateCerts, err = fulcio.GetIntermediates()
			if err != nil {
				return fmt.Errorf("getting Fulcio intermediates: %w", err)
			}
			pubKey, err = cosign.ValidateAndUnpackCert(cert, co)
			if err != nil {
				return err
			}
		} else {
			// Verify certificate with chain
			chain, err := loadCertChainFromFileOrURL(v.cmd.CertChain)
			if err != nil {
				return err
			}
			pubKey, err = cosign.ValidateAndUnpackCertWithChain(cert, chain, co)
			if err != nil {
				return err
			}
		}
		if v.cmd.SCTRef != "" {
			sct, err := os.ReadFile(filepath.Clean(v.cmd.SCTRef))
			if err != nil {
				return fmt.Errorf("reading sct from file: %w", err)
			}
			co.SCT = sct
		}
	}
	co.SigVerifier = pubKey

	ref, err := name.ParseReference(v.image, v.cmd.NameOptions...)
	if err != nil {
		return fmt.Errorf("parsing reference: %w", err)
	}
	ref, err = sign.GetAttachedImageRef(ref, v.cmd.Attachment, ociremoteOpts...)
	if err != nil {
		return fmt.Errorf("resolving attachment type %s for image %s: %w", v.cmd.Attachment, v.image, err)
	}
	verified, bundleVerified, err := cosign.VerifyImageSignatures(ctx, ref, co)
	if err != nil {
		if strings.Contains(err.Error(), "no signatures found") {
			v.result.Payload = err.Error()
			return nil
		}
		return err
	}
	if bundleVerified || co.RekorClient != nil {
		v.result.TLogVerified = true
	}
	for _, sig := range verified {
		cert, err := sig.Cert()
		if err == nil && cert != nil {
			ce := cosign.CertExtensions{Cert: cert}
			sub := ""
			if sans := cryptoutils.GetSubjectAlternateNames(cert); len(sans) > 0 {
				sub = sans[0]
			}
			// fmt.Printf("Certificate subject: %s\n", sub)
			v.result.CertificateSubject = sub
			if issuerURL := ce.GetIssuer(); issuerURL != "" {
				// fmt.Printf("Certificate issuer URL: %s\n", issuerURL)
				v.result.CertificateIssuer = issuerURL
			}
			if githubWorkflowTrigger := ce.GetCertExtensionGithubWorkflowTrigger(); githubWorkflowTrigger != "" {
				// fmt.Printf("GitHub Workflow Trigger: %s\n", githubWorkflowTrigger)
				v.result.GithubWorkflowTrigger = githubWorkflowTrigger
			}
			if githubWorkflowSha := ce.GetExtensionGithubWorkflowSha(); githubWorkflowSha != "" {
				// fmt.Printf("GitHub Workflow SHA: %s\n", githubWorkflowSha)
				v.result.GithubWorkflowSha = githubWorkflowSha
			}
			if githubWorkflowName := ce.GetCertExtensionGithubWorkflowName(); githubWorkflowName != "" {
				// fmt.Printf("GitHub Workflow Name: %s\n", githubWorkflowName)
				v.result.GithubWorkflowName = githubWorkflowName
			}
			if githubWorkflowRepository := ce.GetCertExtensionGithubWorkflowRepository(); githubWorkflowRepository != "" {
				// fmt.Printf("GitHub Workflow Repository: %s\n", githubWorkflowRepository)
				v.result.GithubWorkflowRepository = githubWorkflowRepository
			}
			if githubWorkflowRef := ce.GetCertExtensionGithubWorkflowRef(); githubWorkflowRef != "" {
				// fmt.Printf("GitHub Workflow Ref: %s\n", githubWorkflowRef)
				v.result.GithubWorkflowRef = githubWorkflowRef
			}
		}

		p, err := sig.Payload()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching payload: %v", err)
			return err
		}
		// fmt.Printf("Payload: %v\n", string(p))
		v.result.Payload = string(p)
	}
	return nil
}

func (v *Validator) Result() *ImageResult {
	return v.result
}

func keylessVerification(keyRef string, sk bool) bool {
	if keyRef != "" {
		return false
	}
	if sk {
		return false
	}
	return true
}

func loadCertChainFromFileOrURL(path string) ([]*x509.Certificate, error) {
	pems, err := blob.LoadFileOrURL(path)
	if err != nil {
		return nil, err
	}
	certs, err := cryptoutils.LoadCertificatesFromPEM(bytes.NewReader(pems))
	if err != nil {
		return nil, err
	}
	return certs, nil
}

func loadCertFromFileOrURL(path string) (*x509.Certificate, error) {
	pems, err := blob.LoadFileOrURL(path)
	if err != nil {
		return nil, err
	}
	return loadCertFromPEM(pems)
}

func loadCertFromPEM(pems []byte) (*x509.Certificate, error) {
	var out []byte
	out, err := base64.StdEncoding.DecodeString(string(pems))
	if err != nil {
		// not a base64
		out = pems
	}

	certs, err := cryptoutils.UnmarshalCertificatesFromPEM(out)
	if err != nil {
		return nil, err
	}
	if len(certs) == 0 {
		return nil, errors.New("no certs found in pem file")
	}
	return certs[0], nil
}

func shouldVerifySCT(ignoreSCT bool, keyRef string, sk bool) bool {
	if keyRef != "" {
		return false
	}
	if sk {
		return false
	}
	if ignoreSCT {
		return false
	}
	return true
}
