package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cnrancher/hangar/pkg/image/manifest"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type viewProvenanceOpts struct {
	arch   string
	os     string
	output string

	tlsVerify bool
	autoYes   bool
}

type viewProvenanceCmd struct {
	*baseCmd
	*viewProvenanceOpts
}

func newViewProvenanceCmd() *viewProvenanceCmd {
	cc := &viewProvenanceCmd{
		viewProvenanceOpts: new(viewProvenanceOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "provenance",
		Short:   "View image SLSA Provenance",
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
	flags.StringVarP(&cc.output, "output", "o", "", "output filename (default: STDOUT)")
	flags.StringVarP(&cc.arch, "override-arch", "", "", "use ARCH instead of the architecture of the machine for choosing images")
	flags.StringVarP(&cc.os, "override-os", "", "", "use OS instead of the running OS for choosing images")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false,
		"answer yes automatically (used in shell script)")

	return cc
}

func (cc *viewProvenanceCmd) run(args []string) error {
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
	b, err := inspector.Provenance(signalContext)
	if err != nil {
		return err
	}
	// No-pretty output
	// fmt.Print(string(b))

	m := map[string]any{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return fmt.Errorf("failed to unmarshal provenance data: %w", err)
	}
	b, err = json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal provenance data: %w", err)
	}
	if cc.output != "" {
		err := utils.CheckFileExistsPrompt(signalContext, cc.output, cc.autoYes)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(cc.output, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %q: %w", cc.output, err)
		}
		_, err = f.WriteString(string(b))
		if err != nil {
			return fmt.Errorf("failed to write %q: %w", cc.output, err)
		}
		logrus.Infof("SLSA Provenance output to %q", cc.output)
		return nil
	}
	fmt.Print(string(b))
	fmt.Println()

	return nil
}
