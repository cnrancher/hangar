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
}

type CopierOption struct {
	Options      *imagecopy.Options
	RetryOptions *retry.Options

	SourceRef imagetypes.ImageReference
	DestRef   imagetypes.ImageReference
}

func NewCopier(o *CopierOption) *Copier {
	c := &Copier{
		source:      o.SourceRef,
		destination: o.DestRef,

		options:      o.Options,
		retryOptions: o.RetryOptions,
	}

	return c
}

func (c *Copier) Copy(ctx context.Context) ([]byte, error) {
	var (
		m []byte
	)
	policy, err := signature.DefaultPolicy(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy context: %w", err)
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
