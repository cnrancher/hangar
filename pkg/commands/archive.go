package commands

import (
	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type archiveCmd struct {
	*baseCmd
}

func newArchiveCmd() *archiveCmd {
	cc := &archiveCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "archive",
		Short: "Action for Hangar archive file",
		Long:  "",
		Example: `
# Show images in archive file:
hangar archive ls -f SAVED_ARCHIVE.zip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, cmdconfig.DefaultProvider)
			if cc.baseCmd.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("debug output enabled")
				logrus.Debugf("%v", utils.PrintObject(cmdconfig.Get("")))
			}
			return nil
		},
	})

	// flags := cc.baseCmd.cmd.Flags()

	addCommands(cc.cmd,
		newArchiveLsCmd(),
		newArchiveMergeCmd(),
		newArchiveExportCmd(),
	)
	return cc
}
