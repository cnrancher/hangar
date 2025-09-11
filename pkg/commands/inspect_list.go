package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type inspectListOpts struct {
	file      string
	report    string
	format    string
	jobs      int
	timeout   time.Duration
	failed    string
	registry  string
	project   string
	autoYes   bool
	tlsVerify bool
}

type inspectListCmd struct {
	*baseCmd
	*inspectListOpts
}

func newInspectListCmd() *inspectListCmd {
	cc := &inspectListCmd{
		inspectListOpts: new(inspectListOpts),
	}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "inspect-list -f IMAGE_LIST.txt",
		Aliases: []string{},
		Short:   "Inspect multiple container images by image-list file",
		Long:    "",
		Example: `# Inspect image list file:
hangar inspect-list --file=image-list.txt --report=inspect-report.txt`,
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
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
			if err := run(h); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.file, "file", "f", "", "image list file")
	flags.SetAnnotation("file", cobra.BashCompFilenameExt, []string{"txt"})
	flags.SetAnnotation("file", cobra.BashCompOneRequiredFlag, []string{""})
	flags.StringVarP(&cc.report, "report", "r", "", "inspect report filename (default: inspect-report.[FORMAT])")
	flags.StringVarP(&cc.format, "format", "", "", "inspect report format (json/yaml/csv/txt)")
	flags.StringVarP(&cc.registry, "registry", "", "", "override the registry in image list")
	flags.StringVarP(&cc.project, "project", "", "", "override the project in image list")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, inspect images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when inspect each images")
	flags.StringVarP(&cc.failed, "failed", "o", "inspect-failed.txt", "file name of the inspect failed image list")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")

	return cc
}

func (cc *inspectListCmd) prepareHangar() (hangar.Hangar, error) {
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
	if !cc.tlsVerify {
		sysCtx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!cc.tlsVerify)
		sysCtx.OCIInsecureSkipTLSVerify = !cc.tlsVerify
	}

	policy, err := cc.getPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	s, err := hangar.NewInspector(&hangar.InspectorOpts{
		CommonOpts: hangar.CommonOpts{
			Images:              images,
			Arch:                []string{},
			OS:                  []string{},
			Variant:             nil,
			CopyProvenance:      false,
			Timeout:             cc.timeout,
			Workers:             cc.jobs,
			FailedImageListName: cc.failed,
			SystemContext:       sysCtx,
			Policy:              policy,
			RemoveSignatures:    true,
			SigstorePrivateKey:  "",
			SigstorePublicKey:   "",
		},

		ReportFile:   cc.report,
		ReportFormat: cc.format,
		AutoYes:      cc.autoYes,
		Registry:     cc.registry,
		Project:      cc.project,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create inspector: %v", err)
	}

	return s, nil
}
