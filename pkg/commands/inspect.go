package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/image/manifest"
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
		Use:     "inspect IMAGE_REFERENCE",
		Aliases: []string{"i"},
		Short:   "Inspect provides basic functions of 'skopeo inspect' to inspect image manifest",
		Long:    "",
		Example: `# Inspect image manifest:
hangar inspect [image-reference]

# Inspect RAW docker image maniefest:
hangar inspect docker://docker.io/cnrancher/hangar:latest --raw`,
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
	flags.StringVarP(&cc.arch, "override-arch", "", "", "use ARCH instead of the architecture of the machine for choosing images")
	flags.StringVarP(&cc.os, "override-os", "", "", "use OS instead of the running OS for choosing images")
	flags.StringVarP(&cc.variant, "override-variant", "", "", "use VARIANT instead of the running variant for choosing images")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.raw, "raw", "", false, "output raw manifest")
	flags.BoolVarP(&cc.config, "config", "", false, "output raw configuration")

	return cc
}

func (cc *inspectCmd) run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("image reference not provided")
	}
	switch {
	case strings.HasPrefix(args[0], "docker:"):
	case strings.HasPrefix(args[0], "docker-daemon:"):
	case strings.HasPrefix(args[0], "docker-archive:"):
	case strings.HasPrefix(args[0], "oci:"):
	case strings.HasPrefix(args[0], "dir:"):
	default:
		args[0] = fmt.Sprintf("docker://%s", args[0])
		logrus.Warnf("Image reference protocol not provided, use 'docker' as default: %v", args[0])
	}

	ctx := signalContext
	inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
		ReferenceName: args[0],
		SystemContext: &types.SystemContext{
			ArchitectureChoice:          cc.arch,
			OSChoice:                    cc.os,
			VariantChoice:               cc.variant,
			OCIInsecureSkipTLSVerify:    !cc.tlsVerify,
			DockerInsecureSkipTLSVerify: types.NewOptionalBool(!cc.tlsVerify),
		},
	})
	if err != nil {
		return err
	}
	defer inspector.Close()

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
