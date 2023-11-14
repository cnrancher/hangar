package copy

import (
	"context"
	"fmt"

	"github.com/containers/common/pkg/retry"
	imagecopy "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	imagetypes "github.com/containers/image/v5/types"
)

type Copier struct {
	source      imagetypes.ImageReference
	destination imagetypes.ImageReference

	options      *imagecopy.Options
	retryOptions *retry.Options

	policy *signature.Policy
}

type CopierOption struct {
	Options      *imagecopy.Options
	RetryOptions *retry.Options

	SourceRef imagetypes.ImageReference
	DestRef   imagetypes.ImageReference

	Policy *signature.Policy
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
	policyContext, err := signature.NewPolicyContext(c.policy)
	if err != nil {
		return nil, fmt.Errorf("copy: failed to create policy context: %w", err)
	}
	err = retry.IfNecessary(ctx, func() error {
		var err error
		m, err = imagecopy.Image(
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
