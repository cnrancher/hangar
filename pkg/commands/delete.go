package commands

import (
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/image/manifest"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type deleteCmd struct {
	*baseCmd

	tlsVerify bool
	autoYes   bool
}

func newDeleteCmd() *deleteCmd {
	cc := &deleteCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "delete IMAGE_NAME",
		Aliases: []string{},
		Short:   "Delete image from the registry server",
		Long:    "",
		Example: `hangar delete IMAGE_NAME`,
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cc.run(args); err != nil {
				return err
			}

			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")

	return cc
}

func (cc *deleteCmd) run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("image reference not provided")
	}
	if !strings.HasPrefix(args[0], "docker:") {
		args[0] = fmt.Sprintf("docker://%s", args[0])
	}

	ctx := signalContext
	inspector, err := manifest.NewInspector(&manifest.InspectorOption{
		ReferenceName: args[0],
		SystemContext: &types.SystemContext{
			OCIInsecureSkipTLSVerify:    !cc.tlsVerify,
			DockerInsecureSkipTLSVerify: types.NewOptionalBool(!cc.tlsVerify),
		},
	})
	if err != nil {
		return err
	}
	defer inspector.Close()
	// Ensure image exists.
	if _, _, err := inspector.Raw(ctx); err != nil {
		return err
	}
	var s string
	fmt.Printf("Image %q will be deleted! proceed? [y/N] ", args[0])
	if cc.autoYes {
		fmt.Println("y")
	} else {
		if _, err := utils.Scanf(ctx, "%s", &s); err != nil {
			return err
		}
		if len(s) == 0 || s[0] != 'y' && s[0] != 'Y' {
			return fmt.Errorf("abort by user")
		}
	}

	if err := inspector.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	logrus.Infof("delete %q", args[0])
	return nil
}
