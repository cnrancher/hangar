package commands

import (
	"context"
	"time"

	"github.com/cnrancher/hangar/pkg/signal"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
)

var (
	signalContext context.Context = signal.SetupSignalContext()

	defaultUserAgent string = "hangar/" + utils.Version
)

type baseCmd struct {
	*baseOpts
	cmd *cobra.Command
}

func newBaseCmd(cmd *cobra.Command) *baseCmd {
	return &baseCmd{cmd: cmd, baseOpts: &globalOpts}
}

type baseOpts struct {
	debug          bool   // Enable debug output
	policyPath     string // Path to a signature verification policy file
	insecurePolicy bool   // Use an "allow everything" signature verification policy
}

var globalOpts = baseOpts{}

func (cc *baseCmd) getCommand() *cobra.Command {
	return cc.cmd
}

func (cc *baseCmd) newSystemContext() *types.SystemContext {
	ctx := &types.SystemContext{
		DockerRegistryUserAgent: defaultUserAgent,
	}
	return ctx
}

func (cc *baseCmd) getPolicy() (*signature.Policy, error) {
	var policy *signature.Policy // This could be cached across calls in baseCmd.
	var err error
	if cc.insecurePolicy {
		policy = &signature.Policy{
			Default: []signature.PolicyRequirement{
				signature.NewPRInsecureAcceptAnything(),
			},
			Transports: make(map[string]signature.PolicyTransportScopes),
		}
	} else if cc.policyPath == "" {
		policy, err = signature.DefaultPolicy(nil)
	} else {
		policy, err = signature.NewPolicyFromFile(cc.policyPath)
	}
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// getPolicyContext returns a *signature.PolicyContext based on baseCmd.
func (cc *baseCmd) getPolicyContext() (*signature.PolicyContext, error) {
	policy, err := cc.getPolicy()
	if err != nil {
		return nil, err
	}
	return signature.NewPolicyContext(policy)
}

func (cc *baseCmd) ctxWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	var (
		ctx                       = signalContext
		cancel context.CancelFunc = func() {}
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	return ctx, cancel
}

type cmder interface {
	getCommand() *cobra.Command
}

func addCommands(root *cobra.Command, commands ...cmder) {
	for _, command := range commands {
		cmd := command.getCommand()
		if cmd == nil {
			continue
		}
		root.AddCommand(cmd)
	}
}
