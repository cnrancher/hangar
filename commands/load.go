package commands

import (
	"fmt"
	"time"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/hangar"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type loadCmd struct {
	*baseCmd

	arch           []string
	os             []string
	source         string
	destination    string
	failed         string
	repoType       string
	defaultProject string
	jobs           int
	harborHttps    bool
	timeout        time.Duration
	project        string
	tlsVerify      bool
}

func newLoadCmd() *loadCmd {
	cc := &loadCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "load -s SAVED_ARCHIVE.zip -d REGISTRY_SERVER",
		Short: "Load images from zip archive created by 'save' command onto registry server",
		Long:  "",
		Example: `
hangar load \
	--source SAVED_ARCHIVE.zip \
	--destination REGISTRY_URL \
	--arch amd64,arm64 \
	--os linux`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, cmdconfig.DefaultProvider)
			if cc.baseCmd.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("debug output enabled")
				logrus.Debugf("%v", utils.PrintObject(cmdconfig.Get("")))
			}

			if err := cc.run(); err != nil {
				return err
			}

			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringArrayVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringArrayVarP(&cc.os, "os", "", []string{"linux", "windows"}, "OS list of images")
	flags.StringVarP(&cc.source, "source", "s", "", "saved archive filename")
	flags.StringVarP(&cc.destination, "destination", "d", "", "destination registry url")
	flags.StringVarP(&cc.failed, "failed", "o", "load-failed.txt", "file name of the load failed image list")
	flags.StringVarP(&cc.repoType, "repo-type", "", "", "repository type, can be 'harbor'")
	flags.StringVarP(&cc.defaultProject, "default-project", "", "library", "default project name")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when save each images")
	flags.StringVarP(&cc.project, "project", "", "", "override all destination image projects")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")

	addCommands(cc.cmd, newLoadValidateCmd())

	return cc
}

func (cc *loadCmd) run() error {
	if cc.source == "" {
		return fmt.Errorf("source file not provided, use '--source' to provide the archive file")
	}
	if cc.destination == "" {
		return fmt.Errorf("destination registry URL not provided, use '--destination' to provide the registry")
	}

	l, err := hangar.NewLoader(&hangar.LoaderOpts{
		CommonOpts: hangar.CommonOpts{
			Images:    nil,
			Arch:      cc.arch,
			OS:        cc.os,
			Variant:   nil,
			Timeout:   cc.timeout,
			TlsVerify: cc.tlsVerify,
		},

		DestinationRegistry: cc.destination,
		DestinationProject:  cc.project,
		SharedBlobDirPath:   "", // Use the default shared blob dir path.
		ArchiveName:         cc.source,
	})
	if err != nil {
		return fmt.Errorf("failed to create loader: %v", err)
	}

	err = l.Run(signalContext)
	if err != nil {
		return err
	}
	return err
}
