package commands

import (
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/hangar"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type saveCmd struct {
	*baseCmd

	file        string
	arch        []string
	os          []string
	source      string
	destination string
	failed      string
	compress    string
	jobs        int
	timeout     time.Duration
	tlsVerify   bool
}

func newSaveCmd() *saveCmd {
	cc := &saveCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "save -f IMAGE_LIST.txt -d SAVED_ARCHIVE.tar.gz",
		Short: "Save images from registry server into local tarball archive",
		Long:  "",
		Example: `
hangar save \
	-f IMAGE_LIST.txt \
	--arch amd64,arm64 \
	--os linux \
	-d SAVED_ARCHIVE.tar.gz`,
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
	flags.StringVarP(&cc.destination, "destination", "d", "", "file name of the output saved images")
	flags.StringVarP(&cc.failed, "failed", "o", "save-failed.txt", "file name of the save failed image list")
	flags.StringVarP(&cc.compress, "compress", "c", "gzip", "compress format, can be 'gzip' or 'tar'")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when save each images")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")

	return cc
}

func (cc *saveCmd) run() {
	if cc.file == "" {
		logrus.Fatalf("image list not provided, use '--file' to specify the image list file")
	}

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

	s := hangar.NewSaver(&hangar.SaverOpts{
		CommonOpts: hangar.CommonOpts{
			Images:  images,
			Arch:    cc.arch,
			OS:      cc.os,
			Variant: nil,
			Timeout: cc.timeout,
			Workers: cc.jobs,
		},

		SourceRegistry:    cc.source,
		SharedBlobDirPath: "", // Use the default shared blob dir path.
		ArchiveName:       cc.destination,
	})

	err = s.Run(signalContext)
	if err != nil {
		logrus.Error(err)
	}
}
