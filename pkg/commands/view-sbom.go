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

type viewSBOMOpts struct {
	arch string
	os   string

	tlsVerify bool
	autoYes   bool
}

type viewSBOMCmd struct {
	*baseCmd
	*viewSBOMOpts
}

func newViewSBOMCmd() *viewSBOMCmd {
	cc := &viewSBOMCmd{
		viewSBOMOpts: new(viewSBOMOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "sbom",
		Short:   "View image SBOM data",
		Example: ``,
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cc.run(args)
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.arch, "override-arch", "", "", "use ARCH instead of the architecture of the machine for choosing images")
	flags.StringVarP(&cc.os, "override-os", "", "", "use OS instead of the running OS for choosing images")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false,
		"answer yes automatically (used in shell script)")

	return cc
}

func (cc *viewSBOMCmd) run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("image reference not provided")
	}
	var image string
	switch {
	case strings.HasPrefix(args[0], "docker:"):
		image = strings.TrimPrefix(args[0], "docker://")
	case strings.HasPrefix(args[0], "docker-daemon:") || strings.HasPrefix(args[0], "docker-archive:") ||
		strings.HasPrefix(args[0], "oci:") || strings.HasPrefix(args[0], "dir:"):
		logrus.Errorf("Unsupported protocol provided, only 'docker://' supported")
		return fmt.Errorf("unsupported protocol %v", args[0])
	default:
		image = args[0]
	}

	refName, err := getImageAttestationReference(signalContext, image, cc.os, cc.arch, cc.tlsVerify)
	if err != nil {
		return fmt.Errorf("failed to get image attestation digest: %w", err)
	}
	inspector, err := manifest.NewInspector(signalContext, &manifest.InspectorOption{
		ReferenceName: refName,
		SystemContext: &types.SystemContext{
			OCIInsecureSkipTLSVerify:    !cc.tlsVerify,
			DockerInsecureSkipTLSVerify: types.NewOptionalBool(!cc.tlsVerify),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create inspector: %w", err)
	}
	defer inspector.Close()
	b, err := inspector.SBOM(signalContext)
	if err != nil {
		return err
	}
	// No-pretty output
	// fmt.Print(string(b))

	m := map[string]any{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return fmt.Errorf("failed to unmarshal sbom data: %w", err)
	}
	b, err = json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sbom data: %w", err)
	}
	fmt.Print(string(b))
	fmt.Println()

	return nil
}
