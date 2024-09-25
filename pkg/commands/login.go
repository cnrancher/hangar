package commands

import (
	"os"
	"time"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/common/pkg/auth"
	commonFlag "github.com/containers/common/pkg/flag"
	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type loginCmd struct {
	*baseCmd

	loginOpts    auth.LoginOptions
	tlsVerify    commonFlag.OptionalBool // Require HTTPS and verify certificates
	timeout      time.Duration
	retryOptions retry.Options
}

func newLoginCmd() *loginCmd {
	cc := &loginCmd{}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "login registry-url",
		Short:   "Login to registry server",
		Example: "  hangar login docker.io",
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := cc.baseCmd.ctxWithTimeout(cc.timeout)
			defer cancel()
			cc.loginOpts.Stdin = os.Stdin
			cc.loginOpts.Stdout = os.Stdout
			cc.loginOpts.AcceptUnspecifiedRegistry = true
			cc.loginOpts.AcceptRepositories = true
			sys := cc.baseCmd.newSystemContext()
			if cc.tlsVerify.Present() {
				sys.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!cc.tlsVerify.Value())
			}
			return retry.IfNecessary(ctx, func() error {
				errCh := make(chan error)
				go func() {
					// Use go routine to avoid block when SIGINT.
					errCh <- auth.Login(ctx, sys, &cc.loginOpts, args)
				}()
				select {
				case err := <-errCh:
					return err
				case <-ctx.Done():
					return ctx.Err()
				}
			}, &cc.retryOptions)
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.DurationVarP(&cc.timeout, "timeout", "", 0, "login timeout")
	flags.IntVar(&cc.retryOptions.MaxRetry, "retry-times", 3, "the number of times to possibly retry")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")
	flags.AddFlagSet(auth.GetLoginFlags(&cc.loginOpts))

	return cc
}
