package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar"
	signv2 "github.com/cnrancher/hangar/pkg/image/sign_v2"
	"github.com/cnrancher/hangar/pkg/utils"
	commonFlag "github.com/containers/common/pkg/flag"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type signValidateV2Opts struct {
	file       string
	arch       []string
	os         []string
	registry   string
	project    string
	failed     string
	jobs       int
	timeout    time.Duration
	reportFile string
	format     string
	autoYes    bool
	tlsVerify  commonFlag.OptionalBool

	publicKey             string
	rekorURL              string
	offline               bool
	ignoreTlog            bool
	validateManifestIndex bool
	certIdentity          string
	certOidcIssuer        string
}

type signValidateV2Cmd struct {
	*baseCmd
	*signValidateV2Opts

	report *signv2.Report
}

func newSignValidateV2Cmd() *signValidateV2Cmd {
	cc := &signValidateV2Cmd{
		signValidateV2Opts: new(signValidateV2Opts),
		report:             signv2.NewReport(),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "validate -f IMAGE_LIST.txt",
		Short: "Validate the signed images with cosign sigstore public key",
		Long:  ``,
		Example: `# Validate the signed images by sigstore public key file.
hangar sign validate \
	--file IMAGE_LIST.txt \
	--key cosign.pub \
	--arch amd64,arm64 \
	--os linux`,
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
			logrus.Infof("Validating images in %q with sigstore public key %q",
				cc.file, cc.publicKey)
			if err := validate(h); err != nil {
				return err
			}
			if err := cc.saveReport(); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.file, "file", "f", "", "image list file")
	flags.SetAnnotation("file", cobra.BashCompFilenameExt, []string{"txt"})
	flags.SetAnnotation("file", cobra.BashCompOneRequiredFlag, []string{""})
	flags.StringSliceVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringSliceVarP(&cc.os, "os", "", []string{"linux"}, "OS list of images")
	flags.StringVarP(&cc.registry, "registry", "", "", "override all image registry URL in image list")
	flags.StringVarP(&cc.project, "project", "", "", "override all image projects in image list")
	flags.StringVarP(&cc.failed, "failed", "o", "sign-failed.txt", "file name of the sign failed image list")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when validate each images")
	flags.StringVarP(&cc.reportFile, "report", "r", "sign-validate-report.[FORMAT]", "sign validate report output file")
	flags.StringVarP(&cc.format, "format", "", "json",
		fmt.Sprintf("output report format (available: %v)", strings.Join(signv2.AvailableFormats, ",")))
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")

	flags.StringVarP(&cc.publicKey, "key", "k", "",
		"path to the cosign public key file")
	flags.SetAnnotation("key", cobra.BashCompFilenameExt, []string{"key", "pub"})
	flags.StringVar(&cc.rekorURL, "rekor-url", DefaultRekorURL,
		"address of rekor STL server")
	flags.BoolVar(&cc.offline, "offline", false,
		"only allow offline verification")
	flags.BoolVar(&cc.ignoreTlog, "insecure-ignore-tlog", false,
		"ignore transparency log verification, to be used when an artifact signature has not been uploaded to the transparency log. Artifacts "+
			"cannot be publicly verified when not included in a log")
	flags.BoolVarP(&cc.validateManifestIndex, "validate-manifest-index", "", true,
		"validate cosign sigstore signature of the manifest index")
	flags.StringVar(&cc.certIdentity, "certificate-identity", "",
		"The identity expected in a valid Fulcio certificate. Valid values include email address, DNS names, IP addresses, and URIs. Must be set for keyless flows.")
	flags.StringVar(&cc.certOidcIssuer, "certificate-oidc-issuer", "",
		"The OIDC issuer expected in a valid Fulcio certificate, e.g. https://token.actions.githubusercontent.com or https://oauth2.sigstore.dev/auth. Must be set for keyless flows.")

	return cc
}

func (cc *signValidateV2Cmd) prepareHangar() (hangar.Hangar, error) {
	if cc.file == "" {
		return nil, fmt.Errorf("image list file not provided, use '--file' option to specify the image list file")
	}
	if cc.publicKey == "" {
		if cc.certIdentity == "" || cc.certOidcIssuer == "" {
			return nil, fmt.Errorf("sigstore public key file not provided, " +
				"use '--key' option to specify the pub-key file")
		}
	}
	// if cc.registry == "" {
	// 	return nil, fmt.Errorf("registry URL not provided")
	// }
	if cc.debug {
		logrus.Debugf("Debug mode enabled, force worker number to 1")
		cc.jobs = 1
	} else {
		if cc.jobs > utils.MaxWorkerNum || cc.jobs < utils.MinWorkerNum {
			logrus.Warnf("Invalid worker num: %v, set to 1", cc.jobs)
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
	if cc.publicKey != "" {
		if _, err := os.Stat(cc.publicKey); err != nil {
			return nil, fmt.Errorf("failed to get status of file %q: %w",
				cc.publicKey, err)
		}
	}
	s, err := hangar.NewSignerV2(&hangar.Signerv2Opts{
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
			SigstorePublicKey:   cc.publicKey,
		},

		PublicKey:             cc.publicKey,
		IgnoreTlog:            cc.ignoreTlog,
		CertIdentity:          cc.certIdentity,
		CertOidcIssuer:        cc.certOidcIssuer,
		InsecureSkipTLSVerify: !cc.tlsVerify.Value(),
		ValidateManifestIndex: cc.validateManifestIndex,
		Report:                cc.report,
		Registry:              cc.registry,
		Project:               cc.project,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %v", err)
	}
	logrus.Infof("Arch List: [%v]", strings.Join(cc.arch, ","))
	logrus.Infof("OS List: [%v]", strings.Join(cc.os, ","))
	logrus.Infof("Output Format: [%v]", cc.format)

	return s, nil
}

func (cc *signValidateV2Cmd) saveReport() error {
	var reportFile string
	switch strings.ToLower(cc.format) {
	case signv2.FormatJSON, signv2.FormatYAML, signv2.FormatCSV:
		reportFile = strings.ReplaceAll(cc.reportFile, "[FORMAT]", cc.format)
	default:
		return fmt.Errorf("unsupported output format: %v", cc.format)
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
	case signv2.FormatJSON:
		b, err = json.MarshalIndent(cc.report, "", "  ")
		if err != nil {
			return fmt.Errorf("report marshal json failed: %w", err)
		}
		_, err = f.Write(b)
		if err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
	case signv2.FormatYAML:
		b, err = yaml.Marshal(cc.report)
		if err != nil {
			return fmt.Errorf("report marshal yaml failed: %w", err)
		}
		_, err = f.Write(b)
		if err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
	case signv2.FormatCSV:
		if err = cc.report.WriteCSV(f); err != nil {
			return fmt.Errorf("report write csv failed: %w", err)
		}
	default:
		return fmt.Errorf("unrecognized output format: %v", cc.format)
	}

	logrus.Infof("Scan report output to %q", reportFile)
	return nil
}
