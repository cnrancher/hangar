package commands

import (
	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type loadValidateCmd struct {
	*loadCmd
}

func newLoadValidateCmd(opts *loadOpts) *loadValidateCmd {
	cc := &loadValidateCmd{
		loadCmd: &loadCmd{
			loadOpts: opts,
		},
	}
	cc.loadCmd.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "validate -s SAVED_ARCHIVE.zip -d REGISTRY_SERVER",
		Short: "Validate the loaded images, ensure images were loaded to registry server",
		Long:  "",
		Example: `
hangar load validate \
	--file IMAGE_LIST.txt \
	--source SAVED_ARCHIVE.zip \
	--destination REGISTRY_URL \
	--arch amd64,arm64 \
	--os linux`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, cmdconfig.DefaultProvider)
			if cc.baseCmd.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("debug output enabled")
				logrus.Debugf("%v", utils.PrintObject(cmdconfig.Get("")))
			}
			h, err := cc.prepareHangar()
			if err != nil {
				return err
			}
			if err := validate(h); err != nil {
				return err
			}
			return nil
		},
	})

	return cc
}
