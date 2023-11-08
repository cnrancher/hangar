package commands

import (
	"time"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type saveValidateCmd struct {
	*saveCmd
}

func newSaveValidateCmd(opts *saveOpts) *saveValidateCmd {
	cc := &saveValidateCmd{
		saveCmd: &saveCmd{
			saveOpts: opts,
		},
	}
	cc.saveCmd.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "validate -f IMAGE_LIST.txt -d SAVED_ARCHIVE.zip",
		Short: "Validate the saved images, ensure images were saved into archive file",
		Long:  "",
		Example: `
hangar save validate \
	-f IMAGE_LIST.txt \
	-d SAVED_ARCHIVE.zip`,
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

	flags := cc.saveCmd.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.file, "file", "f", "", "image list file")
	flags.StringArrayVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringArrayVarP(&cc.os, "os", "", []string{"linux", "windows"}, "OS list of images")
	flags.StringVarP(&cc.source, "source", "s", "", "override the source registry in image list")
	flags.StringVarP(&cc.destination, "destination", "d", "saved-images.zip", "file name of the output saved images")
	flags.StringVarP(&cc.failed, "failed", "o", "save-failed.txt", "file name of the save failed image list")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, validate images parallelly")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when save each images")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	return cc
}
