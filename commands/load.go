package commands

import (
	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type loadCmd struct {
	*baseCmd

	source         string
	destination    string
	failed         string
	repoType       string
	defaultProject string
	jobs           int
	harborHttps    bool
}

func newLoadCmd() *loadCmd {
	cc := &loadCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "load -s SAVED_ARCHIVE.tar.gz -d REGISTRY_SERVER",
		Short:   "Load images from tarball created by 'save' command onto registry server",
		Long:    "",
		Example: ``,
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

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.source, "source", "s", "", "saved tarball filename")
	flags.StringVarP(&cc.destination, "destination", "d", "", "destination registry url")
	flags.StringVarP(&cc.failed, "failed", "o", "load-failed.txt", "file name of the load failed image list")
	flags.StringVarP(&cc.repoType, "repo-type", "", "", "repository type, can be 'harbor'")
	flags.StringVarP(&cc.defaultProject, "default-project", "", "library", "default project name")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly")

	addCommands(cc.cmd, newLoadValidateCmd())

	return cc
}

func (cc *loadCmd) run() {
}
