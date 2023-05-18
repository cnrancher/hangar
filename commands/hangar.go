package commands

import (
	"os"

	"github.com/spf13/cobra"
)

func Execute(args []string) {
	hangarCmd := newHangarCmd()
	hangarCmd.addCommands()
	hangarCmd.cmd.SetArgs(args)

	_, err := hangarCmd.cmd.ExecuteC()
	if err != nil {
		os.Exit(1)
	}
}

type hangarCmd struct {
	baseCmd
}

func newHangarCmd() *hangarCmd {
	cc := &hangarCmd{}

	cc.baseCmd.cmd = &cobra.Command{
		Use:   "hangar",
		Short: "Hangar usage",
		Long: `Hangar is a tool for mirror/copy multi-arch container images from the public
registry server to your registry server with manifest list support.
Besides, it also support other container-image related operations such as image
list generation according to Rancher KDM data and chart repositories.

Documents of this tool: https://github.com/cnrancher/hangar#docs
`,
		Run: func(cmd *cobra.Command, args []string) {
			// show help message only
			cmd.Help()
		},
	}
	cc.cmd.CompletionOptions = cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	}
	cc.cmd.Version = getVersion()
	cc.cmd.SilenceUsage = true

	cc.cmd.PersistentFlags().BoolP("debug", "", false, "enable debug output")
	cc.cmd.PersistentFlags().BoolP("tls-verify", "", true, "enable https tls verify")

	return cc
}

func (cc *hangarCmd) getCommand() *cobra.Command {
	return cc.cmd
}

func (cc *hangarCmd) addCommands() {
	addCommands(
		cc.cmd,
		newVersionCmd(),
		newMirrorCmd(),
		newMirrorValidateCmd(),
		newSaveCmd(),
		newLoadCmd(),
		newLoadValidateCmd(),
		newGenerateListCmd(),
		newConvertListCmd(),
		newSyncCmd(),
		newCompressCmd(),
		newDecompressCmd(),
	)
}
