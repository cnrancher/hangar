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

type loadOpts struct {
	file           string
	arch           []string
	os             []string
	source         string
	sourceRegistry string
	destination    string
	failed         string
	repoType       string
	jobs           int
	timeout        time.Duration
	project        string
	skipLogin      bool
	tlsVerify      commonFlag.OptionalBool
}

type loadCmd struct {
	*baseCmd
	*loadOpts
}

func newLoadCmd() *loadCmd {
	cc := &loadCmd{
		loadOpts: new(loadOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "load -s SAVED_ARCHIVE.zip -d REGISTRY_SERVER",
		Short: "Load images from zip archive created by 'save' command to registry server",
		Long: `Load images from zip archive created by 'save' command to registry server.

The load command will create Harbor V2 projects for destination registry automatically.
`,
		Example: `# Load images from SAVED_ARCHIVE.zip to REGISTRY SERVER.
hangar load \
	--file IMAGE_LIST.txt \
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

			h, err := cc.prepareHangar()
			if err != nil {
				return err
			}
			if err := run(h); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.baseCmd.cmd.PersistentFlags()
	flags.StringVarP(&cc.file, "file", "f", "", "image list file (optional: load all images from archive if not provided)")
	flags.SetAnnotation("file", cobra.BashCompFilenameExt, []string{"txt"})
	flags.StringSliceVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringSliceVarP(&cc.os, "os", "", []string{"linux"}, "OS list of images")
	flags.StringVarP(&cc.source, "source", "s", "", "saved archive filename")
	flags.SetAnnotation("source", cobra.BashCompFilenameExt, []string{"zip"})
	flags.SetAnnotation("source", cobra.BashCompOneRequiredFlag, []string{""})
	flags.StringVarP(&cc.sourceRegistry, "source-registry", "", "", "override the source registry of image list")
	flags.StringVarP(&cc.destination, "destination", "d", "", "destination registry url")
	flags.SetAnnotation("destination", cobra.BashCompOneRequiredFlag, []string{""})
	flags.StringVarP(&cc.failed, "failed", "o", "load-failed.txt", "file name of the load failed image list")
	flags.SetAnnotation("failed", cobra.BashCompFilenameExt, []string{"txt"})
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number,copy images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when save each images")
	flags.StringVarP(&cc.project, "project", "", "", "override all destination image projects")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")

	flags.BoolVarP(&cc.skipLogin, "skip-login", "", false,
		"skip check the destination registry is logged in (used in shell script)")

	addCommands(
		cc.cmd,
		newLoadValidateCmd(cc.loadOpts),
	)
	return cc
}

func (cc *loadCmd) prepareHangar() (hangar.Hangar, error) {
	if cc.source == "" {
		return nil, fmt.Errorf("source file not provided, use '--source' to provide the archive file")
	}
	if cc.destination == "" {
		return nil, fmt.Errorf("destination registry URL not provided, use '--destination' to provide the registry")
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

	var images []string
	if cc.file != "" {
		file, err := os.Open(cc.file)
		if err != nil {
			return nil, fmt.Errorf("failed to open %q: %v", cc.file, err)
		}
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
	}

	sysCtx := cc.baseCmd.newSystemContext()
	if cc.tlsVerify.Present() {
		sysCtx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!cc.tlsVerify.Value())
		sysCtx.OCIInsecureSkipTLSVerify = !cc.tlsVerify.Value()
	}

	if !cc.skipLogin {
		// Only check whether the destination registry needs login.
		if err := prepareLogin(
			signalContext,
			map[string]bool{cc.destination: true},
			utils.CopySystemContext(sysCtx),
		); err != nil {
			return nil, err
		}
	}

	policy, err := cc.getPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	l, err := hangar.NewLoader(&hangar.LoaderOpts{
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

		SourceRegistry:      cc.sourceRegistry,
		DestinationRegistry: cc.destination,
		DestinationProject:  cc.project,
		SharedBlobDirPath:   "", // Use the default shared blob dir path.
		ArchiveName:         cc.source,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create loader: %v", err)
	}
	logrus.Infof("Arch List: [%v]", strings.Join(cc.arch, ","))
	logrus.Infof("OS List: [%v]", strings.Join(cc.os, ","))

	return l, nil
}
