package commands

import (
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type mirrorValidateCmd struct {
	*mirrorCmd
}

func newMirrorValidateCmd(opts *mirrorOpts) *mirrorValidateCmd {
	cc := &mirrorValidateCmd{
		mirrorCmd: &mirrorCmd{
			mirrorOpts: opts,
		},
	}
	cc.mirrorCmd.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "validate -f IMAGE_LIST.txt -d DESTINATION_REGISTRY",
		Short: "Ensure the images were mirrored correctly",
		Long:  ``,
		Example: `
hangar mirror validate \
	--file IMAGE_LIST.txt \
	--source SOURCE_REGISTRY \
	--destination DESTINATION_REGISTRY \
	--arch amd64,arm64 \
	--os linux`,
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
			if cc.sigstorePassphraseFile != "" || cc.sigstorePrivateKey != "" {
				logrus.Warnf("The 'hangar mirror validate' command does not support" +
					" validating the image sigstore signature.")
				logrus.Warnf("Use 'hangar sign validate' command to validate the" +
					" signature with sigstore public key file instead.")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := cc.mirrorCmd.prepareHangar()
			if err != nil {
				return err
			}
			if err := validate(h); err != nil {
				return err
			}
			return nil
		},
	})

	usageFunc := cc.cmd.UsageFunc()
	cc.cmd.SetUsageFunc(func(c *cobra.Command) error {
		c.PersistentFlags().MarkHidden("sigstore-private-key")
		c.PersistentFlags().MarkHidden("sigstore-passphrase-file")
		for p := c.Parent(); p != nil; p = p.Parent() {
			p.PersistentFlags().MarkHidden("sigstore-private-key")
			p.PersistentFlags().MarkHidden("sigstore-passphrase-file")
		}
		usageFunc(cc.cmd)
		return nil
	})

	return cc
}
