package commands

import (
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type syncValidateCmd struct {
	*syncCmd
}

func newSyncValidateCmd(opts *syncOpts) *syncValidateCmd {
	cc := &syncValidateCmd{
		syncCmd: &syncCmd{
			syncOpts: opts,
		},
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "validate -f IMAGE_LIST.txt -d SAVED_ARCHIVE.zip",
		Short: "Validate the sync (append) images, ensure images were saved into archive file",
		Long:  "",
		Example: `
hangar sync validate \
	--file IMAGE_LIST.txt \
	--source SOURCE_REGISTRY \
	--destination SAVED_ARCHIVE.zip \
	--arch amd64,arm64 \
	--os linux`,
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
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
