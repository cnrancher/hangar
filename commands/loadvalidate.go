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
	-s SAVED_ARCHIVE.zip \
	-d REGISTRY_URL`,
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

	flags := cc.loadCmd.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.file, "file", "f", "", "image list file (optional: validate all images in archive if not provided)")
	flags.StringSliceVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringSliceVarP(&cc.os, "os", "", []string{"linux", "windows"}, "OS list of images")
	flags.StringVarP(&cc.source, "source", "s", "", "saved archive filename")
	flags.StringVarP(&cc.destination, "destination", "d", "", "destination registry url")
	flags.StringVarP(&cc.failed, "failed", "o", "load-failed.txt", "file name of the load failed image list")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, validate images parallelly")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	return cc
}
