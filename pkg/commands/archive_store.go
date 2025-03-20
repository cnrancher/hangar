package commands

import (
	"github.com/spf13/cobra"
)

type archiveStoreCmd struct {
	*baseCmd
}

func newArchiveStoreCmd() *archiveStoreCmd {
	cc := &archiveStoreCmd{}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "store",
		Short:   "store files in Hangar archive file",
		Aliases: []string{"s"},
		Long:    "",
		Example: `# Store chart file into hangar archive file
hangar archive store chart --help

# Store files into hangar archive file
hangar archive store file --help
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cc.cmd.Help()
		},
	})

	addCommands(cc.cmd,
		newStoreChartCmd(),
		newStoreFileCmd())
	return cc
}
