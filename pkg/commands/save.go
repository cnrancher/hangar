package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

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
		PreRun: func(cmd *cobra.Command, args []string) {
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := cc.prepareHangar()
			if err != nil {
				return err
			}
			if err := utils.CheckFileExistsPrompt(signalContext, cc.destination, cc.autoYes); err != nil {
				return err
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
		logrus.Debugf("Debug mode enabled, force worker number to 1")
		cc.jobs = 1
	} else {
		if cc.jobs > utils.MaxWorkerNum || cc.jobs < utils.MinWorkerNum {
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
			// Hangar does not support to save image signatures into
			// local archive file yet.
			RemoveSignatures:   true,
			SigstorePrivateKey: "",
			SigstorePublicKey:  "",
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
