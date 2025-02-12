package signv2

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/generate"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
)

func (s *Signer) NameOptions() []name.Option {
	var nameOpts []name.Option
	if s.insecureSkipTLSVerify {
		nameOpts = append(nameOpts, name.Insecure) // Allow HTTP registry
	}
	return nameOpts
}

func (s *Signer) ClientOpts(ctx context.Context) []ociremote.Option {
	opts := []ociremote.Option{
		ociremote.WithRemoteOptions(s.GetRegistryClientOpts(ctx)...),
	}
	return opts
}

func (s *Signer) GetRegistryClientOpts(ctx context.Context) []remote.Option {
	opts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithUserAgent(utils.DefaultUserAgent()),
	}

	switch {
	case s.authConfig.Username != "" && s.authConfig.Password != "":
		opts = append(opts, remote.WithAuth(&authn.Basic{
			Username: s.authConfig.Username,
			Password: s.authConfig.Password,
		}))
	case s.authConfig.RegistryToken != "":
		opts = append(opts, remote.WithAuth(&authn.Bearer{
			Token: s.authConfig.RegistryToken,
		}))
	default:
		opts = append(opts, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	}

	tlsConfig, err := s.getTLSConfig()
	if err == nil {
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = tlsConfig
		opts = append(opts, remote.WithTransport(tr))
	}
	tlsConfig.InsecureSkipVerify = s.insecureSkipTLSVerify

	return opts
}

func (s *Signer) getTLSConfig() (*tls.Config, error) {
	var tlsConfig tls.Config
	tlsConfig.InsecureSkipVerify = s.insecureSkipTLSVerify
	return &tlsConfig, nil
}

func (s *Signer) keyOptions() *options.KeyOpts {
	return &options.KeyOpts{
		KeyRef:                   s.key,
		PassFunc:                 generate.GetPass,
		RekorURL:                 s.rekorURL,
		OIDCIssuer:               s.oidcIssuer,
		OIDCClientID:             s.oidcClientID,
		OIDCClientSecret:         "",
		OIDCProvider:             s.oidcProvider,
		InsecureSkipFulcioVerify: false,
	}
}
