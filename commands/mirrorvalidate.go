package commands

import (
	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type mirrorValidateCmd struct {
	*mirrorCmd
}

func newMirrorValidateCmd() *mirrorValidateCmd {
	cc := &mirrorValidateCmd{mirrorCmd: &mirrorCmd{}}
	cc.mirrorCmd.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "validate -f IMAGE_LIST.txt -d DESTINATION_REGISTRY",
		Short: "Ensure the images were mirrored correctly",
		Long:  ``,
		Example: `
hangar mirror validate \
	--file IMAGE_LIST.txt \
	--source SOURCE_REGISTRY \
	--destination DESTINATION_REGISTRY`,
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

	flags := cc.mirrorCmd.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.file, "file", "f", "", "image list file")
	flags.StringArrayVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringArrayVarP(&cc.os, "os", "", []string{"linux", "windows"}, "OS list of images")
	flags.StringVarP(&cc.source, "source", "s", "", "override the source registry in image list")
	flags.StringVarP(&cc.destination, "destination", "d", "", "specify the destination image registry")
	flags.StringVarP(&cc.failed, "failed", "o", "mirror-failed.txt", "file name of the mirror failed image list")
	flags.IntP("jobs", "j", 1, "worker number, copy images parallelly")
	flags.StringVarP(&cc.repoType, "repo-type", "", "", "destination registry type, can be 'harbor'")
	flags.BoolVarP(&cc.harborHttps, "harbor-https", "", true, "use https when create harbor project")

	return cc
}
