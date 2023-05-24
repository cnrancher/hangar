package commands

import (
	"strings"

	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type mirrorValidateCmd struct {
	mirrorCmd
}

func newMirrorValidateCmd() *mirrorValidateCmd {
	cc := &mirrorValidateCmd{
		mirrorCmd: mirrorCmd{
			registriesSet: make(map[string]struct{}),
		},
	}

	cc.cmd = &cobra.Command{
		Use:     "mirror-validate",
		Short:   "Validate the mirrored images",
		Long:    `Validate the mirrored images`,
		Example: "  hangar mirror-validate -f MIRROR_IMAGE_LIST.txt -s SOURCE -d DESTINATION",
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, config.DefaultProvider)

			if config.GetBool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
			}

			if err := cc.mirrorCmd.baseCmd.selfCheckDependencies(); err != nil {
				return err
			}
			if err := cc.mirrorCmd.setupFlags(); err != nil {
				return err
			}
			if err := cc.mirrorCmd.baseCmd.processSkopeoLogin(); err != nil {
				return err
			}
			if err := cc.mirrorCmd.processImageList(); err != nil {
				return err
			}
			cc.mirrorCmd.baseCmd.prepareWorker()
			cc.run()
			cc.mirrorCmd.finish()
			return nil
		},
	}

	cc.cmd.Flags().StringP("file", "f", "", "image list file (should be 'mirror' format)")
	cc.cmd.Flags().StringP("arch", "a", "amd64,arm64", "architecture list of images, separate with ','")
	cc.cmd.Flags().StringP("os", "", "linux,windows", "OS list of images, separate with ','")
	cc.cmd.Flags().StringP("source", "s", "", "override the source registry defined in image list")
	cc.cmd.Flags().StringP("destination", "d", "", "override the destination registry defined in image list")
	cc.cmd.Flags().StringP("failed", "o", "mirror-validate-failed.txt", "file name of the mirror validate failed image list")
	cc.cmd.Flags().IntP("jobs", "j", 1, "worker number, concurrent mode if larger than 1, max 20")
	cc.cmd.Flags().StringP("default-project", "", "library", "project name (also called 'namespace') when destination image project is empty")

	return cc
}

func (cc *mirrorValidateCmd) run() {
	if cc.baseCmd.workerChan == nil {
		panic("workerChan not initialized")
	}
	for i, v := range cc.listSpec {
		m := mirror.NewMirror(&mirror.MirrorOptions{
			Source:      v.source,
			Destination: v.destination,
			Tag:         v.tag,
			ArchList:    strings.Split(config.GetString("arch"), ","),
			OsList:      strings.Split(config.GetString("os"), ","),
			Line:        v.line,
			Mode:        mirror.MODE_MIRROR_VALIDATE,
			ID:          i + 1,
		})
		cc.baseCmd.workerChan <- m
	}
}
