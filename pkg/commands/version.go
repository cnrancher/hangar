package commands

import (
	"fmt"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type versionCmd struct {
	*baseCmd
}

func newVersionCmd() *versionCmd {
	cc := &versionCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "version",
		Short:   "Show version",
		Example: "  hangar version",
		PreRun: func(cmd *cobra.Command, args []string) {
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("hangar version %s\n", getVersion())
		},
	})

	return cc
}

func (cc *versionCmd) getCommand() *cobra.Command {
	return cc.cmd
}

func getVersion() string {
	if utils.GitCommit != "" {
		return fmt.Sprintf("%s - %s", utils.Version, utils.GitCommit)
	}
	return utils.Version
}
