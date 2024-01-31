package hangar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/image/source"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

// signObject is the object sending to worker pool when signing image
type signObject struct {
	image   string
	source  *source.Source
	timeout time.Duration
	id      int
}

type Signer struct {
	*common

	// sigstorePublicKey is the file path of the sigstore public key
	sigstorePublicKey string

	// See containers/image exactRepository signedIdentity
	exactRepository string

	// Override the registry of all images to be signed
	Registry string
	// Override the project of all images to be signed
	Project string
}

// Signer implements functions of Hangar interface.
var _ Hangar = &Signer{}

type SignerOpts struct {
	CommonOpts

	ExactRepository string
	Registry        string
	Project         string
}

func NewSigner(o *SignerOpts) (*Signer, error) {
	s := &Signer{
		sigstorePublicKey: o.SigstorePublicKey,
		exactRepository:   o.ExactRepository,
		Registry:          o.Registry,
		Project:           o.Project,
	}
	var err error
	s.common, err = newCommon(&o.CommonOpts)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Signer) sign(ctx context.Context) {
	s.common.initErrorHandler(ctx)
	s.common.initWorker(ctx, s.worker)
	for i, line := range s.common.images {
		var (
			object *signObject
			err    error
		)
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", line)
			continue
		}
		object = &signObject{
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

func (s *Signer) worker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*signObject)
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
	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("Signing [%v]", obj.source.ReferenceNameWithoutTransport())
	err = obj.source.Sign(signContext, &source.SignOptions{
		SigstorePrivateKey: s.common.sigstorePrivateKey,
		SigstorePassphrase: s.common.sigstorePassphrase,
		Set:                s.common.imageSpecSet,
		Policy:             s.common.policy,
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
func (s *Signer) Run(ctx context.Context) error {
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

func (s *Signer) Validate(ctx context.Context) error {
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

func (s *Signer) validate(ctx context.Context) {
	s.common.initErrorHandler(ctx)
	s.initWorker(ctx, s.validateWorker)
	for i, line := range s.common.images {
		var (
			object *signObject
			err    error
		)
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", line)
			continue
		}
		object = &signObject{
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

func (s *Signer) validateWorker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*signObject)
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
	err = obj.source.ValidateSignature(
		validateContext,
		s.sigstorePublicKey,
		s.exactRepository,
		s.imageSpecSet,
	)
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
	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("PASS: [%v]", obj.source.ReferenceNameWithoutTransport())
}
