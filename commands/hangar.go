package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/pkg/docker/config"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func Execute(args []string) error {
	hangarCmd := newHangarCmd()
	hangarCmd.addCommands()
	hangarCmd.cmd.SetArgs(args)

	_, err := hangarCmd.cmd.ExecuteC()
	if err != nil {
		if signalContext.Err() != nil {
			return signalContext.Err()
		} else {
			return err
		}
	}
	return nil
}

type hangarCmd struct {
	*baseCmd
}

func newHangarCmd() *hangarCmd {
	cc := &hangarCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use: "hangar",
		Long: `Hangar supports container image copy operations between registry servers.
Docs: https://github.com/cnrancher/hangar#docs
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	})
	cc.cmd.Version = getVersion()
	cc.cmd.SilenceUsage = true
	cc.cmd.SilenceErrors = true

	flags := cc.cmd.PersistentFlags()
	flags.BoolVarP(&cc.baseCmd.debug, "debug", "", false, "enable debug output")
	flags.BoolVar(&cc.baseCmd.insecurePolicy, "insecure-policy", false, "run Hangar without policy check")

	return cc
}

func (cc *hangarCmd) getCommand() *cobra.Command {
	return cc.cmd
}

func (cc *hangarCmd) addCommands() {
	addCommands(
		cc.cmd,
		newVersionCmd(),
		newLoginCmd(),
		newLogoutCmd(),
		newMirrorCmd(),
		newSaveCmd(),
		newLoadCmd(),
		newSyncCmd(),
		newArchiveCmd(),
		newInspectCmd(),
		newConvertListCmd(),
		newGenerateListCmd(),
	)
}

// run executes hangar.Run()
func run(h hangar.Hangar) error {
	if err := h.Run(signalContext); err != nil {
		// Error occurred while run, save copy failed image to file.
		if err := h.SaveFailedImages(); err != nil {
			return err
		}
		return err
	}
	return nil
}

// validate executes hangar.Validate()
func validate(h hangar.Hangar) error {
	if err := h.Validate(signalContext); err != nil {
		// Error occurred while validate, save validate failed image to file.
		if err := h.SaveFailedImages(); err != nil {
			return err
		}
		return err
	}
	return nil
}

func prepareLogin(
	ctx context.Context,
	registrySet map[string]bool,
	sysCtx *types.SystemContext,
) error {
	if sysCtx == nil {
		sysCtx = &types.SystemContext{}
	}
	for registry := range registrySet {
		authConfig, err := config.GetCredentials(sysCtx, registry)
		if err != nil {
			return fmt.Errorf("failed to get credential of registry %q: %w",
				registry, err)
		}
		if authConfig.Password != "" {
			continue
		}

		logrus.Infof("Logging into %q", registry)
		err = retry.IfNecessary(ctx, func() error {
			errCh := make(chan error)
			go func() {
				// Use go routine to avoid block when SIGINT.
				errCh <- auth.Login(ctx, sysCtx, &auth.LoginOptions{
					Stdin:                     os.Stdin,
					Stdout:                    os.Stdout,
					AcceptUnspecifiedRegistry: true,
				}, []string{registry})
			}()
			select {
			case err := <-errCh:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}, &retry.Options{
			MaxRetry: 3,
			Delay:    time.Microsecond * 100,
		})
		if err != nil {
			return fmt.Errorf("failed to login to %q: %w", registry, err)
		}
	}
	return nil
}
