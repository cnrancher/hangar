package commands

import (
	"github.com/cnrancher/hangar/pkg/archive"
	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type loadValidateCmd struct {
	loadCmd
}

func newLoadValidateCmd() *loadValidateCmd {
	cc := &loadValidateCmd{}

	cc.cmd = &cobra.Command{
		Use:     "load-validate",
		Short:   "Validate the loaded images",
		Long:    `Validate the loaded images`,
		Example: "  hangar load-validate -s SAVED_FILE.tar.gz -d REGISTRY_URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, config.DefaultProvider)

			if config.GetBool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
			}

			if err := cc.loadCmd.baseCmd.selfCheckDependencies(
				checkDockerSkopeo); err != nil {
				return err
			}
			if err := cc.loadCmd.setupFlags(); err != nil {
				return err
			}
			if cc.compressFormat != archive.CompressFormatDirectory {
				err := cc.loadCmd.baseCmd.prepareImageCacheDirectory()
				if err != nil {
					return err
				}
			}
			if err := cc.loadCmd.decompressTarball(); err != nil {
				return err
			}
			if err := cc.loadCmd.baseCmd.processDockerLogin(); err != nil {
				return err
			}
			if err := cc.loadCmd.prepareMirrorers(); err != nil {
				return err
			}
			cc.loadCmd.baseCmd.prepareWorker()
			cc.run()
			cc.loadCmd.baseCmd.finish()

			return nil
		},
	}

	cc.cmd.Flags().StringP("source", "s", "", "saved file to load validate "+
		"(need to use '--compress' to specify the file format if not gzip)")
	cc.cmd.Flags().StringP("destination", "d", "", "destination regitry")
	cc.cmd.Flags().StringP("failed", "o", "load-validate-failed.txt",
		"file name of the validate failed image list")
	cc.cmd.Flags().StringP("compress", "", "gzip",
		"compress format, can be 'gzip', 'zstd', or 'dir'")
	cc.cmd.Flags().IntP("jobs", "j", 1,
		"worker number, concurrent mode if larger than 1, max 20")
	cc.cmd.Flags().StringP("default-project", "", "library",
		"project name (also called 'namespace') when destination image project is empty")

	return cc
}

func (cc *loadValidateCmd) run() {
	for _, m := range cc.loadCmd.mirrorers {
		m.Mode = mirror.MODE_LOAD_VALIDATE
		cc.loadCmd.baseCmd.workerChan <- m
	}
}
