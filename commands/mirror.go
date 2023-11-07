package commands

import (
	"bufio"
	"context"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/hangar"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type mirrorCmd struct {
	*baseCmd

	file           string
	arch           []string
	os             []string
	source         string
	destination    string
	failed         string
	jobs           int
	repoType       string
	defaultProject string
	harborHttps    bool
	timeout        time.Duration
	tlsVerify      bool

	sourceProject      string
	destinationProject string
}

func newMirrorCmd() *mirrorCmd {
	cc := &mirrorCmd{}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "mirror -f IMAGE_LIST.txt -d DESTINATION_REGISTRY",
		Short: "Mirror images between registry servers",
		Long:  ``,
		Example: `
hangar mirror \
	--file IMAGE_LIST.txt \
	--source SOURCE_REGISTRY \
	--destination DESTINATION_REGISTRY \
	--arch amd64,arm64,s390x \
	--os linux`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, cmdconfig.DefaultProvider)
			if cc.baseCmd.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("debug output enabled")
				logrus.Debugf("%v", utils.PrintObject(cmdconfig.Get("")))
			}

			cc.run()

			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.file, "file", "f", "", "image list file")
	flags.StringArrayVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringArrayVarP(&cc.os, "os", "", []string{"linux", "windows"}, "OS list of images")
	flags.StringVarP(&cc.source, "source", "s", "", "override the source registry in image list")
	flags.StringVarP(&cc.destination, "destination", "d", "", "specify the destination image registry")
	flags.StringVarP(&cc.failed, "failed", "o", "mirror-failed.txt", "file name of the mirror failed image list")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly")
	flags.StringVarP(&cc.repoType, "repo-type", "", "", "destination registry type, can be 'harbor'")
	flags.BoolVarP(&cc.harborHttps, "harbor-https", "", true, "use https when create harbor project")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when mirror each images")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")

	flags.StringVarP(&cc.sourceProject, "source-project", "", "",
		"override all source image projects")
	flags.StringVarP(&cc.destinationProject, "destination-project", "", "",
		"override all destination image projects")

	addCommands(cc.cmd, newMirrorValidateCmd())

	return cc
}

func (cc *mirrorCmd) run() {
	if cc.file == "" {
		logrus.Fatalf("file not provided")
	}
	// if cc.destination == "" {
	// 	logrus.Fatalf("destination registry URL not provided")
	// }

	file, err := os.Open(cc.file)
	if err != nil {
		logrus.Fatalf("failed to open %q: %v", cc.file, err)
	}

	images := []string{}
	sc := bufio.NewScanner(file)
	sc.Split(bufio.ScanLines)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			continue
		}
		images = append(images, l)
	}
	if err := file.Close(); err != nil {
		logrus.Fatalf("failed to close %q: %v", cc.file, err)
	}

	m := hangar.NewMirrorer(&hangar.MirrorerOpts{
		CommonOpts: hangar.CommonOpts{
			Images:  images,
			Arch:    cc.arch,
			OS:      cc.os,
			Variant: nil, // TODO: support variants
			Timeout: cc.timeout,
			Workers: cc.jobs,
		},

		SourceRegistry:      cc.source,
		SourceProject:       cc.sourceProject,
		DestinationRegistry: cc.destination,
		DestinationProject:  cc.destinationProject,
	})

	err = m.Run(context.Background())
	if err != nil {
		logrus.Error(err)
	}
}
