package commands

import (
	"encoding/json"
	"fmt"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/manifest"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type inspectCmd struct {
	*baseCmd

	arch      string
	os        string
	variant   string
	raw       bool
	config    bool
	tlsVerify bool
}

func newInspectCmd() *inspectCmd {
	cc := &inspectCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "inspect IMAGR_REFERENCE",
		Short: "Inspect provides basic functions of 'skopeo inspect' to inspect image manifest",
		Long:  "",
		Example: `
# Inspect RAW docker image maniefest:
  hangar inspect docker://docker.io/cnrancher/hangar:latest --raw`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, cmdconfig.DefaultProvider)
			if cc.baseCmd.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("debug output enabled")
				logrus.Debugf("%v", utils.PrintObject(cmdconfig.Get("")))
			}
			if err := cc.run(args); err != nil {
				return err
			}

			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.arch, "override-arch", "", "", "use ARCH instead of the architecture of the machine for choosing images")
	flags.StringVarP(&cc.os, "override-os", "", "", "use OS instead of the running OS for choosing images")
	flags.StringVarP(&cc.variant, "override-variant", "", "", "use VARIANT instead of the running variant for choosing images")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.raw, "raw", "", false, "output raw manifest or configuration")
	flags.BoolVarP(&cc.config, "config", "", false, "output configuration")

	return cc
}

func (cc *inspectCmd) run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("image reference not provided")
	}

	ctx := signalContext
	inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
		ReferenceName: args[0],
		SystemContext: &types.SystemContext{
			ArchitectureChoice:          cc.arch,
			OSChoice:                    cc.os,
			VariantChoice:               cc.variant,
			OCIInsecureSkipTLSVerify:    cc.tlsVerify,
			DockerInsecureSkipTLSVerify: types.NewOptionalBool(cc.tlsVerify),
		},
	})
	if err != nil {
		return err
	}
	switch {
	case cc.config:
		b, err := inspector.Config(ctx)
		if err != nil {
			return err
		}
		fmt.Print(string(b))
	case cc.raw:
		b, _, err := inspector.Raw(ctx)
		if err != nil {
			return err
		}
		fmt.Print(string(b))
	default:
		info, err := inspector.Inspect(ctx)
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(b))
	}

	return nil
}
