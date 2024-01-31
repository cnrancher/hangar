package copy

import (
	"context"
	"fmt"

	"github.com/containers/common/pkg/retry"
	copyv5 "github.com/containers/image/v5/copy"
	signaturev5 "github.com/containers/image/v5/signature"
	typesv5 "github.com/containers/image/v5/types"
)

type Copier struct {
	source      typesv5.ImageReference
	destination typesv5.ImageReference

	options      *copyv5.Options
	retryOptions *retry.Options

	policy *signaturev5.Policy
}

type CopierOption struct {
	Options      *copyv5.Options
	RetryOptions *retry.Options

	SourceRef typesv5.ImageReference
	DestRef   typesv5.ImageReference

	Policy *signaturev5.Policy
}

func NewCopier(o *CopierOption) *Copier {
	c := &Copier{
		source:      o.SourceRef,
		destination: o.DestRef,

		policy:       o.Policy,
		options:      o.Options,
		retryOptions: o.RetryOptions,
	}

	return c
}

func (c *Copier) Copy(ctx context.Context) ([]byte, error) {
	var (
		m []byte
	)
	policyContext, err := signaturev5.NewPolicyContext(c.policy)
	if err != nil {
		return nil, fmt.Errorf("copy: failed to create policy context: %w", err)
	}
	err = retry.IfNecessary(ctx, func() error {
		var err error
		m, err = copyv5.Image(
			ctx,
			policyContext,
			c.destination,
			c.source,
			c.options,
		)
		if err != nil {
			return err
		}
		return nil
	}, c.retryOptions)

	return m, err
}
