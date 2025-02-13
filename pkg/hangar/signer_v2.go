package hangar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	signv2 "github.com/cnrancher/hangar/pkg/image/sign_v2"
	"github.com/cnrancher/hangar/pkg/image/source"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/pkg/docker/config"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/verify"
	"github.com/sirupsen/logrus"
)

// signv2Object is the object sending to worker pool when signing image
type signv2Object struct {
	image   string
	source  *source.Source
	timeout time.Duration
	id      int
}

type SignerV2 struct {
	*common

	// publicKey is the sigstore public key filename (for validate)
	publicKey string

	// privateKey is the sigstore private key filename
	privateKey string

	// tlogUpload uploads to sigstore transparency log server or not
	tlogUpload bool

	// ignoreTlog ignores transparency log server (for validate)
	ignoreTlog bool

	// recordCreationTimestamp records timestamp or not
	recordCreationTimestamp bool

	// rekorURL is the address of rekor STL server
	rekorURL string

	// OIDC provider to be used to issue ID token
	oidcIssuer string

	// oidcClientID is the client ID for application (default sigstore)
	oidcClientID string

	// oidcProvider is the provider to get the OIDC token
	// (spiffe, google, github-actions, filesystem, buildkite-agent)
	oidcProvider string

	// allow HTTP & insecure TLS certificate registry
	insecureSkipTLSVerify bool

	cacheMu         *sync.Mutex
	authConfigCache map[string]*authn.AuthConfig

	// certIdentity is the fulcio certificate (for keyless validate)
	certIdentity string

	// certOidcIssuer is the OIDC issuer of fulcio certificate (for keyless validate)
	certOidcIssuer string

	// signManifestIndex will create a cosign signature for manifest index
	signManifestIndex bool

	// validateManifestIndex will validate the cosign signature of manifest index
	validateManifestIndex bool

	reportMu *sync.Mutex
	report   *signv2.Report

	// Override the registry of all images to be signed
	Registry string
	// Override the project of all images to be signed
	Project string
}

// Signer implements functions of Hangar interface.
var _ Hangar = &SignerV2{}

type Signerv2Opts struct {
	CommonOpts

	// sigstore public key filename
	PublicKey string

	// sigstore private key filename
	PrivateKey string

	// uploads to sigstore transparency log server or not
	TLogUpload bool

	// IgnoreTlog ignores transparency log server (for validate)
	IgnoreTlog bool

	// records timestamp or not
	RecordCreationTimestamp bool

	// rekorURL is the address of rekor STL server
	RekorURL string

	// OIDC provider to be used to issue ID token
	OIDCIssuer string

	// client ID for application (default sigstore)
	OIDCClientID string

	// provider to get the OIDC token
	// (spiffe, google, github-actions, filesystem, buildkite-agent)
	OIDCProvider string

	// allow HTTP & insecure TLS certificate registry
	InsecureSkipTLSVerify bool

	// CertIdentity is the fulcio certificate (for keyless validate)
	CertIdentity string

	// CertOidcIssuer is the OIDC issuer of fulcio certificate (for keyless validate)
	CertOidcIssuer string

	// signManifestIndex will create a cosign signature for manifest index
	SignManifestIndex bool

	// validateManifestIndex will validate the cosign signature of manifest index
	ValidateManifestIndex bool

	Report *signv2.Report

	Registry string
	Project  string
}

func NewSignerV2(o *Signerv2Opts) (*SignerV2, error) {
	s := &SignerV2{
		publicKey:               o.PublicKey,
		privateKey:              o.PrivateKey,
		tlogUpload:              o.TLogUpload,
		recordCreationTimestamp: o.RecordCreationTimestamp,
		rekorURL:                o.RekorURL,
		oidcIssuer:              o.OIDCIssuer,
		oidcClientID:            o.OIDCClientID,
		oidcProvider:            o.OIDCProvider,
		insecureSkipTLSVerify:   o.InsecureSkipTLSVerify,
		certIdentity:            o.CertIdentity,
		certOidcIssuer:          o.CertOidcIssuer,
		signManifestIndex:       o.SignManifestIndex,
		validateManifestIndex:   o.ValidateManifestIndex,
		reportMu:                &sync.Mutex{},
		report:                  o.Report,
		Registry:                o.Registry,
		Project:                 o.Project,

		cacheMu:         &sync.Mutex{},
		authConfigCache: make(map[string]*authn.AuthConfig),
	}
	if s.report == nil {
		s.report = signv2.NewReport()
	}
	var err error
	s.common, err = newCommon(&o.CommonOpts)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SignerV2) sign(ctx context.Context) {
	s.common.initErrorHandler(ctx)
	s.common.initWorker(ctx, s.worker)
	for i, line := range s.common.images {
		var (
			object *signv2Object
			err    error
		)
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", line)
			continue
		}
		object = &signv2Object{
			id:      i + 1,
			image:   line,
			timeout: s.timeout,
		}
		registry := utils.GetRegistryName(line)
		if s.Registry != "" {
			registry = s.Registry
		}
		project := utils.GetProjectName(line)
		if s.Project != "" {
			project = s.Project
		}
		src, err := source.NewSource(&source.Option{
			Type:          types.TypeDocker,
			Registry:      registry,
			Project:       project,
			Name:          utils.GetImageName(line),
			Tag:           utils.GetImageTag(line),
			SystemContext: s.systemContext,
		})
		if err != nil {
			s.handleError(fmt.Errorf("failed to init source image: %w", err))
			s.recordFailedImage(line)
			continue
		}
		object.source = src
		if err = s.handleObject(object); err != nil {
			s.handleError(err)
			s.recordFailedImage(line)
		}
	}
	s.waitWorkers()
}

func (s *SignerV2) worker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*signv2Object)
	if !ok {
		logrus.Errorf("skip object type(%T), data %v", o, o)
		return
	}

	var (
		signContext context.Context
		cancel      context.CancelFunc
		err         error
	)
	if obj.timeout > 0 {
		signContext, cancel = context.WithTimeout(ctx, obj.timeout)
	} else {
		signContext, cancel = context.WithCancel(ctx)
	}
	defer func() {
		cancel()
		if err != nil {
			s.handleError(fmt.Errorf("error occurred when sign [%v]: %w",
				obj.source.ReferenceNameWithoutTransport(), err))
			s.common.recordFailedImage(obj.image)
		}
	}()

	err = obj.source.Init(signContext)
	if err != nil {
		err = fmt.Errorf("failed to init [%v]: %w",
			obj.source.ReferenceName(), err)
		return
	}

	s.cacheMu.Lock()
	var c *authn.AuthConfig
	if p, ok := s.authConfigCache[obj.source.Registry()]; ok {
		c = p
	} else {
		dc, _ := config.GetCredentials(s.systemContext, obj.source.Registry())
		c = &authn.AuthConfig{
			Username:      dc.Username,
			Password:      dc.Password,
			IdentityToken: dc.IdentityToken,
		}
		s.authConfigCache[obj.source.Registry()] = c
	}
	s.cacheMu.Unlock()

	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("Signing [%v]", obj.source.ReferenceNameWithoutTransport())
	err = obj.source.SignV2(signContext, &source.SignV2Options{
		SignerOption: signv2.SignerOption{
			Key:                     s.privateKey,
			Recursive:               false,
			TlogUpload:              s.tlogUpload,
			RecordCreationTimestamp: s.recordCreationTimestamp,
			RekorURL:                s.rekorURL,
			OIDCIssuer:              s.oidcIssuer,
			OIDCClientID:            s.oidcClientID,
			OIDCProvider:            s.oidcProvider,
			InsecureSkipTLSVerify:   s.insecureSkipTLSVerify,
			AuthConfig:              *c,
		},
		SignManifestIndex: s.signManifestIndex,
		Set:               s.common.imageSpecSet,
	})
	if err != nil {
		if errors.Is(err, utils.ErrNoAvailableImage) {
			logrus.WithFields(logrus.Fields{"IMG": obj.id}).
				Warnf("Skip sign image [%v]: %v",
					obj.source.ReferenceNameWithoutTransport(), err)
			err = nil
			return
		}
		err = fmt.Errorf("failed to sign [%v]: %w",
			obj.source.ReferenceName(), err)
		return
	}
}

// Run sign all images in the registry server.
func (s *SignerV2) Run(ctx context.Context) error {
	s.sign(ctx)
	if len(s.failedImageSet) != 0 {
		v := make([]string, 0, len(s.failedImageSet))
		for i := range s.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Sign failed image list: \n%v", strings.Join(v, "\n"))
		return ErrSignFailed
	}
	return nil
}

func (s *SignerV2) Validate(ctx context.Context) error {
	s.validate(ctx)
	if len(s.failedImageSet) != 0 {
		v := make([]string, 0, len(s.failedImageSet))
		for i := range s.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Signature validate failed image list: \n%v",
			strings.Join(v, "\n"))
		return ErrCopyFailed
	}
	return nil
}

func (s *SignerV2) validate(ctx context.Context) {
	s.common.initErrorHandler(ctx)
	s.initWorker(ctx, s.validateWorker)
	for i, line := range s.common.images {
		var (
			object *signv2Object
			err    error
		)
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", line)
			continue
		}
		object = &signv2Object{
			id:      i + 1,
			image:   line,
			timeout: s.timeout,
		}
		registry := utils.GetRegistryName(line)
		if s.Registry != "" {
			registry = s.Registry
		}
		project := utils.GetProjectName(line)
		if s.Project != "" {
			project = s.Project
		}
		src, err := source.NewSource(&source.Option{
			Type:          types.TypeDocker,
			Registry:      registry,
			Project:       project,
			Name:          utils.GetImageName(line),
			Tag:           utils.GetImageTag(line),
			SystemContext: s.systemContext,
		})
		if err != nil {
			s.handleError(fmt.Errorf("failed to init source image: %w", err))
			s.recordFailedImage(line)
			continue
		}
		object.source = src
		if err = s.handleObject(object); err != nil {
			s.handleError(err)
			s.recordFailedImage(line)
		}
	}
	s.waitWorkers()
}

func (s *SignerV2) validateWorker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*signv2Object)
	if !ok {
		logrus.Errorf("skip object type(%T), data %v", o, o)
		return
	}

	var (
		validateContext context.Context
		cancel          context.CancelFunc
		err             error
	)
	if obj.timeout > 0 {
		validateContext, cancel = context.WithTimeout(ctx, obj.timeout)
	} else {
		validateContext, cancel = context.WithCancel(ctx)
	}
	defer func() {
		cancel()
		if err != nil {
			s.handleError(NewError(obj.id, err, obj.source, nil))
			s.common.recordFailedImage(obj.image)
		}
	}()
	err = obj.source.Init(validateContext)
	if err != nil {
		return
	}

	s.cacheMu.Lock()
	var c *authn.AuthConfig
	if p, ok := s.authConfigCache[obj.source.Registry()]; ok {
		c = p
	} else {
		dc, _ := config.GetCredentials(s.systemContext, obj.source.Registry())
		c = &authn.AuthConfig{
			Username:      dc.Username,
			Password:      dc.Password,
			IdentityToken: dc.IdentityToken,
		}
		s.authConfigCache[obj.source.Registry()] = c
	}
	s.cacheMu.Unlock()

	var results []*signv2.ImageResult
	results, err = obj.source.ValidateSignatureV2(validateContext, &source.ValidateV2Options{
		ValidatorOption: signv2.ValidatorOption{
			VerifyCommand: verify.VerifyCommand{
				RegistryOptions: options.RegistryOptions{
					AllowInsecure:     s.insecureSkipTLSVerify,
					AllowHTTPRegistry: s.insecureSkipTLSVerify,
					AuthConfig:        *c,
				},
				CertVerifyOptions: options.CertVerifyOptions{
					CertIdentity:   s.certIdentity,
					CertOidcIssuer: s.certOidcIssuer,
				},
				KeyRef:     s.publicKey,
				RekorURL:   s.rekorURL,
				IgnoreTlog: s.ignoreTlog,
			},
		},
		ValidateManifestIndex: s.validateManifestIndex,
		Set:                   s.common.imageSpecSet,
	})
	if err != nil {
		if errors.Is(err, utils.ErrNoAvailableImage) {
			logrus.WithFields(logrus.Fields{"IMG": obj.id}).
				Warnf("Skip validate image signature [%v]: %v",
					obj.source.ReferenceNameWithoutTransport(), err)
			err = nil
			return
		}
		err = fmt.Errorf("failed to validate signature [%v]: %w",
			obj.source.ReferenceName(), err)
		return
	}
	noSignaturesFound := true
	for i := 0; i < len(results); i++ {
		if results[i].Payload != "no signatures found" {
			noSignaturesFound = false
			break
		}
	}
	if noSignaturesFound {
		logrus.WithFields(logrus.Fields{"IMG": obj.id}).
			Warnf("No signature found of image [%v]", obj.source.ReferenceNameWithoutTransport())
	} else {
		logrus.WithFields(logrus.Fields{"IMG": obj.id}).
			Infof("PASS: [%v]", obj.source.ReferenceNameWithoutTransport())
	}
	s.reportMu.Lock()
	s.report.Append(signv2.NewResult(obj.source.ReferenceNameWithoutTransport(), results))
	s.reportMu.Unlock()
}
