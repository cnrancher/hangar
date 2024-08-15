package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar"
	"github.com/cnrancher/hangar/pkg/image/scan"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type scanOpts struct {
	trivyServerURL   string
	file             string
	arch             []string
	os               []string
	registry         string
	project          string
	failed           string
	jobs             int
	timeout          time.Duration
	cacheDir         string
	dbRepo           string
	javaDBRepo       string
	offline          bool
	skipDBUpdate     bool
	skipJavaDBUpdate bool
	trivyLogOutput   bool
	tlsVerify        bool
	autoYes          bool
	reportFile       string
	scanners         []string
	format           string
}

type scanCmd struct {
	*baseCmd
	*scanOpts

	report *scan.Report
}

func newScanCmd() *scanCmd {
	cc := &scanCmd{
		scanOpts: new(scanOpts),
		report:   scan.NewReport(),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "scan -f IMAGE_LIST.txt",
		Short: "Scan container image vulnerabilities",
		Long:  ``,
		Example: `# Scan images by image list file and output CSV result.
hangar scan \
	--file IMAGE_LIST.txt \
	--format csv \
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
			logrus.Infof("Scanning images in %q", cc.file)
			if err := cc.prepareScanner(); err != nil {
				return err
			}
			if err := run(h); err != nil {
				logrus.Warn(err)
			}
			if err := cc.saveReport(); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.trivyServerURL, "server", "s", "", "trivy server URL (scan as a trivy client mode)")
	flags.StringVarP(&cc.file, "file", "f", "", "image list file")
	flags.SetAnnotation("file", cobra.BashCompFilenameExt, []string{"txt"})
	flags.SetAnnotation("file", cobra.BashCompOneRequiredFlag, []string{""})
	flags.StringSliceVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringSliceVarP(&cc.os, "os", "", []string{"linux"}, "OS list of images")
	flags.StringVarP(&cc.registry, "registry", "", "", "override all image registry URL in image list")
	flags.StringVarP(&cc.project, "project", "", "", "override all image projects in image list")
	flags.StringVarP(&cc.failed, "failed", "o", "scan-failed.txt", "file name of the scan failed image list")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, scan images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when scan each images")
	flags.StringVarP(&cc.cacheDir, "cache", "", utils.TrivyCacheDir(), "trivy database cache directory")
	flags.StringVarP(&cc.dbRepo, "trivy-db-repo", "", scan.DefaultDBRepository,
		"trivy vulnerability database repository")
	flags.StringVarP(&cc.javaDBRepo, "trivy-java-db-repo", "", scan.DefaultJavaDBRepository,
		"trivy java database repository")
	flags.BoolVarP(&cc.offline, "offline-scan", "", false, "scan in offline (air-gapped) mode")
	flags.BoolVarP(&cc.skipDBUpdate, "skip-db-update", "", false, "skip updating trivy vulnerability database")
	flags.BoolVarP(&cc.skipJavaDBUpdate, "skip-java-db-update", "", false, "skip updating trivy java index database")
	flags.BoolVarP(&cc.trivyLogOutput, "trivy-log-output", "", false, "show trivy log (only available in single worker mode)")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")
	flags.StringVarP(&cc.reportFile, "report", "r", "scan-report.[FORMAT]", "scan report output file")
	flags.StringSliceVarP(&cc.scanners, "scanner", "", []string{"vuln"}, "list of scanners (available: vuln,misconfig,secret,license)")
	flags.StringVarP(&cc.format, "format", "", "csv",
		fmt.Sprintf("output report format (available: %v)", strings.Join(scan.AvailableFormats, ",")))

	return cc
}

func (cc *scanCmd) prepareHangar() (hangar.Hangar, error) {
	if cc.trivyServerURL == "" {
		// return nil, fmt.Errorf("trivy server URL not provided, use '--server' option to specify the trivy server URL")
		logrus.Debugf("Start scanning images in local mode.")
	} else {
		logrus.Infof("Scanning image in trivy client mode.")
	}
	if cc.cacheDir == "" {
		return nil, fmt.Errorf("trivy cache directory not specified")
	}
	if cc.file == "" {
		return nil, fmt.Errorf("image list file not provided, use '--file' option to specify the image list file")
	}
	if cc.reportFile == "" {
		return nil, fmt.Errorf("output report file not provided, use '--report' to specify the output file name")
	}
	if len(cc.scanners) == 0 {
		cc.scanners = append(cc.scanners, "vuln")
	} else {
		for _, s := range cc.scanners {
			switch s {
			case "vuln", "misconfig", "secret", "license", "none":
			default:
				return nil, fmt.Errorf("invalid scanner type %q provided", s)
			}
		}
	}
	if cc.format == "" {
		logrus.Warnf("Output report format not specified, set to default: %v",
			scan.FormatJSON)
		cc.format = scan.FormatJSON
	}
	if !slices.Contains(scan.AvailableFormats, cc.format) {
		return nil, fmt.Errorf("invalid output format %q, available [%v]",
			cc.format, strings.Join(scan.AvailableFormats, ","))
	}
	if cc.format == scan.FormatSPDXCSV || cc.format == scan.FormatJSON {
		logrus.Infof("SPDX SBOM output format disables security scanning")
		cc.scanners = []string{"none"}
	}
	if cc.debug {
		logrus.Debugf("Debug mode enabled, force worker number to 1")
		cc.jobs = 1
	} else {
		if cc.jobs > utils.MaxWorkerNum || cc.jobs < utils.MinWorkerNum {
			logrus.Warnf("Invalid worker num: %v, set to 1", cc.jobs)
			cc.jobs = 1
		}
	}
	if cc.trivyLogOutput {
		if cc.jobs > 1 {
			logrus.Warnf("Trivy log output was disabled when worker num larger than 1.")
			cc.trivyLogOutput = false
		} else {
			logrus.Infof("Trivy log output enabled.")
		}
	}

	file, err := os.Open(cc.file)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %v", cc.file, err)
	}
	sysCtx := cc.baseCmd.newSystemContext()
	sysCtx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!cc.tlsVerify)
	sysCtx.OCIInsecureSkipTLSVerify = !cc.tlsVerify

	policy, err := cc.getPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
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

	s, err := hangar.NewScanner(&hangar.ScannerOpts{
		CommonOpts: hangar.CommonOpts{
			Images:              images,
			Arch:                cc.arch,
			OS:                  cc.os,
			Variant:             nil,
			CopyProvenance:      false,
			Timeout:             cc.timeout,
			Workers:             cc.jobs,
			FailedImageListName: cc.failed,
			SystemContext:       sysCtx,
			Policy:              policy,
			SigstorePrivateKey:  "",
			SigstorePassphrase:  nil,
		},

		Report:   cc.report,
		Registry: cc.registry,
		Project:  cc.project,
	})
	if err != nil {
		return nil, fmt.Errorf("hangar.NewScanner failed: %v", err)
	}
	logrus.Infof("Arch List: [%v]", strings.Join(cc.arch, ","))
	logrus.Infof("OS List: [%v]", strings.Join(cc.os, ","))
	logrus.Infof("Scanners: [%v]", strings.Join(cc.scanners, ","))
	logrus.Infof("Output Format: [%v]", cc.format)

	return s, nil
}

func (cc *scanCmd) prepareScanner() error {
	if cc.trivyServerURL != "" {
		u, resp, err := utils.DetectURL(signalContext, cc.trivyServerURL, !cc.tlsVerify)
		if err != nil {
			logrus.Errorf("failed to ping trivy server: %v", err)
			return err
		}
		resp.Body.Close()
		cc.trivyServerURL = u
	}
	// Disable the trivy log output by default.
	scan.InitTrivyLogOutput(cc.debug, !cc.trivyLogOutput)
	// Init trivy database.
	err := scan.InitTrivyDatabase(signalContext, scan.DBOptions{
		TrivyServerURL:        cc.trivyServerURL,
		CacheDirectory:        cc.cacheDir,
		DBRepository:          cc.dbRepo,
		JavaDBRepository:      cc.javaDBRepo,
		SkipUpdateDB:          cc.skipDBUpdate,
		SkipUpdateJavaDB:      cc.skipJavaDBUpdate,
		InsecureSkipTLSVerify: !cc.tlsVerify,
	})
	if err != nil {
		return err
	}
	// Init global scanner.
	err = scan.InitScanner(scan.ScannerOption{
		TrivyServerURL:        cc.trivyServerURL,
		Offline:               cc.offline,
		InsecureSkipTLSVerify: !cc.tlsVerify,
		CacheDirectory:        cc.cacheDir,
		Format:                cc.format,
		Scanners:              cc.scanners,
	})
	if err != nil {
		return err
	}
	return nil
}

func (cc *scanCmd) saveReport() error {
	var reportFile string
	switch cc.format {
	case scan.FormatSPDXJSON:
		reportFile = strings.ReplaceAll(cc.reportFile, "[FORMAT]", "spdx.json")
	case scan.FormatCSV:
		reportFile = strings.ReplaceAll(cc.reportFile, "[FORMAT]", "spdx.csv")
	default:
		reportFile = strings.ReplaceAll(cc.reportFile, "[FORMAT]", cc.format)
	}
	err := utils.CheckFileExistsPrompt(signalContext, reportFile, cc.autoYes)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(reportFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Update time
	cc.report.Time = time.Now()
	logrus.Debugf("Save report time %v", cc.report.Time)
	var b []byte
	switch cc.format {
	case scan.FormatJSON:
		b, err = json.MarshalIndent(cc.report, "", "  ")
		if err != nil {
			return fmt.Errorf("report marshal json failed: %w", err)
		}
		_, err = f.Write(b)
		if err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
	case scan.FormatYAML:
		b, err = yaml.Marshal(cc.report)
		if err != nil {
			return fmt.Errorf("report marshal yaml failed: %w", err)
		}
		_, err = f.Write(b)
		if err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
	case scan.FormatCSV:
		if err = cc.report.WriteCSV(f); err != nil {
			return fmt.Errorf("report write csv failed: %w", err)
		}
	case scan.FormatSPDXJSON:
		b, err = json.MarshalIndent(cc.report, "", "  ")
		if err != nil {
			return fmt.Errorf("report marshal json failed: %w", err)
		}
		_, err = f.Write(b)
		if err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
	case scan.FormatSPDXCSV:
		if err = cc.report.WriteSPDXCSV(f); err != nil {
			return fmt.Errorf("report write csv failed: %w", err)
		}
	default:
		return fmt.Errorf("unrecognized output format: %v", cc.format)
	}

	logrus.Infof("Scan report output to %q", reportFile)
	return nil
}
