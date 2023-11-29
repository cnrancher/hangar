package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/hangar"
	"github.com/cnrancher/hangar/pkg/utils"
	commonFlag "github.com/containers/common/pkg/flag"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type saveOpts struct {
	file        string
	arch        []string
	os          []string
	source      string
	destination string
	failed      string
	jobs        int
	timeout     time.Duration
	tlsVerify   commonFlag.OptionalBool
	autoYes     bool
}

type saveCmd struct {
	*baseCmd
	*saveOpts
}

func newSaveCmd() *saveCmd {
	cc := &saveCmd{
		saveOpts: new(saveOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "save -f IMAGE_LIST.txt -d SAVED_ARCHIVE.zip",
		Short: "Save images from registry server into local archive file",
		Long:  "",
		Example: `
hangar save \
	--file IMAGE_LIST.txt \
	--source SOURCE_REGISTRY \
	--destination SAVED_ARCHIVE.zip \
	--arch amd64,arm64 \
	--os linux`,
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

			if _, err = os.Stat(cc.destination); err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("failed to stat file [%v]: %w",
						cc.destination, err)
				}
			} else {
				fmt.Printf("File %q already exists! Overwrite? [y/N] ", cc.destination)
				if cc.autoYes {
					fmt.Println("y")
				} else {
					var s string
					if _, err = utils.Scanf(signalContext, "%s", &s); err != nil {
						return err
					}
					if len(s) == 0 || s[0] != 'y' && s[0] != 'Y' {
						logrus.Warnf("Abort.")
						return fmt.Errorf("file %q already exists", cc.destination)
					}
				}
			}

			if err := run(h); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.baseCmd.cmd.PersistentFlags()
	flags.StringVarP(&cc.file, "file", "f", "", "image list file")
	flags.SetAnnotation("file", cobra.BashCompFilenameExt, []string{"txt"})
	flags.SetAnnotation("file", cobra.BashCompOneRequiredFlag, []string{""})
	flags.StringSliceVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringSliceVarP(&cc.os, "os", "", []string{"linux"}, "OS list of images")
	flags.StringVarP(&cc.source, "source", "s", "", "override the source registry in image list")
	flags.StringVarP(&cc.destination, "destination", "d", "saved-images.zip", "file name of the output saved images")
	flags.SetAnnotation("destination", cobra.BashCompFilenameExt, []string{"zip"})
	flags.StringVarP(&cc.failed, "failed", "o", "save-failed.txt", "file name of the save failed image list")
	flags.SetAnnotation("failed", cobra.BashCompFilenameExt, []string{"txt"})
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when save each images")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")

	addCommands(
		cc.cmd,
		newSaveValidateCmd(cc.saveOpts),
	)
	return cc
}

func (cc *saveCmd) prepareHangar() (hangar.Hangar, error) {
	if cc.file == "" {
		return nil, fmt.Errorf("image list not provided, use '--file' to specify the image list file")
	}
	if cc.debug {
		logrus.Infof("debug mode enabled, force worker number to 1")
		cc.jobs = 1
	} else {
		if cc.jobs > utils.MAX_WORKER_NUM || cc.jobs < utils.MIN_WORKER_NUM {
			logrus.Warnf("invalid worker num: %v, set to 1", cc.jobs)
			cc.jobs = 1
		}
	}

	file, err := os.Open(cc.file)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %v", cc.file, err)
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
		return nil, fmt.Errorf("failed to close %q: %v", cc.file, err)
	}

	sysCtx := cc.baseCmd.newSystemContext()
	if cc.tlsVerify.Present() {
		sysCtx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!cc.tlsVerify.Value())
		sysCtx.OCIInsecureSkipTLSVerify = !cc.tlsVerify.Value()
	}

	policy, err := cc.getPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	s, err := hangar.NewSaver(&hangar.SaverOpts{
		CommonOpts: hangar.CommonOpts{
			Images:              images,
			Arch:                cc.arch,
			OS:                  cc.os,
			Variant:             nil,
			Timeout:             cc.timeout,
			Workers:             cc.jobs,
			FailedImageListName: cc.failed,
			SystemContext:       sysCtx,
			Policy:              policy,
		},

		SourceRegistry:    cc.source,
		SharedBlobDirPath: "", // Use the default shared blob dir path.
		ArchiveName:       cc.destination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create saver: %v", err)
	}
	logrus.Infof("Arch List: [%v]", strings.Join(cc.arch, ","))
	logrus.Infof("OS List: [%v]", strings.Join(cc.os, ","))

	return s, nil
}
