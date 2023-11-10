package commands

import (
	"os"

	"github.com/cnrancher/hangar/pkg/hangar"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func Execute(args []string) {
	hangarCmd := newHangarCmd()
	hangarCmd.addCommands()
	hangarCmd.cmd.SetArgs(args)

	_, err := hangarCmd.cmd.ExecuteC()
	if err != nil {
		if signalContext.Err() != nil {
			logrus.Error(signalContext.Err())
		} else {
			logrus.Error(err)
		}
		os.Exit(1)
	}
}

type hangarCmd struct {
	*baseCmd
}

func newHangarCmd() *hangarCmd {
	cc := &hangarCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use: "hangar",
		Long: `Hangar is a tool for mirror/copy multi-arch container images from the public
registry server to your registry server with manifest list support.
Besides, it also support other container-image related operations such as image
list generation according to Rancher KDM data and chart repositories.

Documents of this tool: https://github.com/cnrancher/hangar#docs
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	})
	cc.cmd.Version = getVersion()
	cc.cmd.SilenceUsage = true
	cc.cmd.SilenceErrors = true

	cc.cmd.PersistentFlags().BoolVarP(&cc.baseCmd.debug, "debug", "", false, "enable debug output")

	return cc
}

func (cc *hangarCmd) getCommand() *cobra.Command {
	return cc.cmd
}

func (cc *hangarCmd) addCommands() {
	addCommands(
		cc.cmd,
		newVersionCmd(),
		newLoginCmd(),
		newLogoutCmd(),
		newMirrorCmd(),
		newSaveCmd(),
		newLoadCmd(),
		newSyncCmd(),
		newArchiveCmd(),
		newInspectCmd(),
		newConvertListCmd(),
		newGenerateListCmd(),
	)
}

// run executes hangar.Run()
func run(h hangar.Hangar) error {
	if err := h.Run(signalContext); err != nil {
		// Error occurred while run, save copy failed image to file.
		if err := h.SaveFailedImages(); err != nil {
			return err
		}
		return err
	}
	return nil
}

// validate executes hangar.Validate()
func validate(h hangar.Hangar) error {
	if err := h.Validate(signalContext); err != nil {
		// Error occurred while validate, save validate failed image to file.
		if err := h.SaveFailedImages(); err != nil {
			return err
		}
		return err
	}
	return nil
}
