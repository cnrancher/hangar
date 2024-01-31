package commands

import (
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
		PreRun: func(cmd *cobra.Command, args []string) {
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
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
