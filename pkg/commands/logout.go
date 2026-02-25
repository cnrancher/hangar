package commands

import (
	"os"
	"time"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.podman.io/common/pkg/auth"
	commonFlag "go.podman.io/common/pkg/flag"
	"go.podman.io/common/pkg/retry"
	"go.podman.io/image/v5/types"
)

type logoutCmd struct {
	*baseCmd

	logoutOpts   auth.LogoutOptions
	tlsVerify    commonFlag.OptionalBool // Require HTTPS and verify certificates
	timeout      time.Duration
	retryOptions retry.Options
}

func newLogoutCmd() *logoutCmd {
	cc := &logoutCmd{}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "logout registry-url",
		Short:   "Logout from registry server",
		Example: "  hangar logout docker.io",
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
			cc.logoutOpts.Stdout = os.Stdout
			sys := cc.baseCmd.newSystemContext()
			if cc.tlsVerify.Present() {
				sys.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!cc.tlsVerify.Value())
			}
			return retry.IfNecessary(ctx, func() error {
				return auth.Logout(sys, &cc.logoutOpts, args)
			}, &cc.retryOptions)
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.DurationVarP(&cc.timeout, "timeout", "", 0, "logout timeout")
	flags.IntVar(&cc.retryOptions.MaxRetry, "retry-times", 3, "the number of times to possibly retry")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")
	flags.AddFlagSet(auth.GetLogoutFlags(&cc.logoutOpts))

	return cc
}
