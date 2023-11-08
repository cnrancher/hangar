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

func newLoadValidateCmd() *loadValidateCmd {
	cc := &loadValidateCmd{loadCmd: &loadCmd{}}
	cc.loadCmd.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "validate -s SAVED_ARCHIVE.zip -d REGISTRY_SERVER",
		Short: "Validate the loaded images, ensure images were loaded to registry server",
		Long:  "",
		Example: `
hangar load validate \
	-s SAVED_ARCHIVE.zip \
	-d REGISTRY_URL`,
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

	flags := cc.loadCmd.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.source, "source", "s", "", "saved archive filename")
	flags.StringVarP(&cc.destination, "destination", "d", "", "destination registry url")
	flags.StringVarP(&cc.failed, "failed", "o", "load-failed.txt", "file name of the load failed image list")
	flags.StringVarP(&cc.repoType, "repo-type", "", "", "repository type, can be 'harbor'")
	flags.StringVarP(&cc.defaultProject, "default-project", "", "library", "default project name")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")

	return cc
}

func (cc *loadValidateCmd) run() {
}
