package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar"
	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/utils"
	commonFlag "github.com/containers/common/pkg/flag"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	DefaultOIDCIssuerURL = "https://oauth2.sigstore.dev/auth"
	DefaultRekorURL      = "https://rekor.sigstore.dev"
	DefaultFulcioURL     = "https://fulcio.sigstore.dev"
)

type signOpts struct {
	file      string
	arch      []string
	os        []string
	registry  string
	project   string
	failed    string
	jobs      int
	timeout   time.Duration
	skipLogin bool
	tlsVerify commonFlag.OptionalBool
	autoYes   bool

	privateKey              string
	passphraseFile          string
	signManifestIndex       bool
	tlogUpload              bool
	issueCertificate        bool
	signContainerIdentity   string
	recordCreationTimestamp bool
	rekorURL                string
	fulcioURL               string
	oidcIssuer              string
	oidcClientID            string
	oidcProvider            string
}

type signCmd struct {
	*baseCmd
	*signOpts
}

func newSignCmd() *signCmd {
	cc := &signCmd{
		signOpts: new(signOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "sign",
		Short:   "Sign images with cosign sigstore private key",
		Long:    ``,
		Example: `hangar sign --key cosign.key <IMAGE>`,
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
			logrus.Infof("Signing images in %q with sigstore priv-key %q.",
				cc.file, cc.privateKey)
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
	flags.StringSliceVarP(&cc.arch, "arch", "a", []string{"amd64", "arm64"}, "architecture list of images")
	flags.StringSliceVarP(&cc.os, "os", "", []string{"linux"}, "OS list of images")
	flags.StringVarP(&cc.registry, "registry", "", "", "override all image registry URL in image list")
	flags.StringVarP(&cc.project, "project", "", "", "override all image projects in image list")
	flags.StringVarP(&cc.failed, "failed", "o", "scan-failed.txt", "file name of the scan failed image list")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, scan images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when scan each images")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.skipLogin, "skip-login", "", false,
		"skip check the registry is logged in (used in shell script)")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")

	flags.StringVarP(&cc.privateKey, "key", "p", "",
		"path to the private key file, KMS URI or Kubernetes Secret")
	flags.SetAnnotation("key", cobra.BashCompFilenameExt, []string{"key"})
	flags.StringVarP(&cc.passphraseFile, "passphrase-file", "", "",
		"private key passphrase file")
	flags.BoolVarP(&cc.signManifestIndex, "sign-manifest-index", "", true,
		"create cosign sigstore signature for manifest index")
	flags.MarkHidden("sign-manifest-index")
	flags.BoolVar(&cc.tlogUpload, "tlog-upload", true,
		"whether or not to upload to the cosign transparency log server")
	flags.StringVar(&cc.oidcIssuer, "oidc-issuer", DefaultOIDCIssuerURL,
		"OIDC provider to be used to issue ID token")
	flags.StringVar(&cc.oidcClientID, "oidc-client-id", "sigstore",
		"OIDC client ID for application")
	flags.StringVar(&cc.oidcProvider, "oidc-provider", "",
		"Specify the provider to get the OIDC token from (Optional) "+
			"(available: spiffe, google, github-actions, filesystem, buildkite-agent)")
	flags.StringVar(&cc.rekorURL, "rekor-url", DefaultRekorURL,
		"address of rekor STL server")
	flags.StringVar(&cc.fulcioURL, "fulcio-url", DefaultFulcioURL,
		"address of sigstore PKI server")

	addCommands(
		cc.cmd,
		newSignValidateV2Cmd(),
	)

	return cc
}

func (cc *signCmd) prepareHangar() (hangar.Hangar, error) {
	if cc.file == "" {
		return nil, fmt.Errorf("image list file not provided, use '--file' option to specify the image list file")
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

	if !cc.skipLogin {
		// Only check whether the destination registry URL needs login.
		registrySet := cc.getRegistrySet(images)
		if err := prepareLogin(
			signalContext,
			registrySet,
			utils.CopySystemContext(sysCtx),
		); err != nil {
			return nil, err
		}
	}

	if cc.oidcProvider == "" {
		if cc.privateKey == "" {
			return nil, fmt.Errorf("private key file not provided, " +
				"use '--key' option to specify the private key file")
		}
		if cc.passphraseFile != "" {
			b, err := os.ReadFile(cc.passphraseFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %q: %w", cc.passphraseFile, err)
			}
			os.Setenv("COSIGN_PASSWORD", strings.TrimSpace(string(b)))
		} else {
			fmt.Printf("Enter the password of private key: ")
			p, err := utils.ReadPassword(signalContext)
			if err != nil {
				return nil, err
			}
			os.Setenv("COSIGN_PASSWORD", string(p))
		}
	} else {
		logrus.Infof("Using OIDC Provider [%v]", cc.oidcProvider)
	}

	if cc.tlogUpload {
		var s string
		fmt.Printf("COSIGN transparency log upload (https://github.com/sigstore/fulcio/blob/main/docs/ctlog.md) enabled, proceed? [y/N] ")
		if cc.autoYes {
			fmt.Println("y")
		} else {
			if _, err := utils.Scanf(signalContext, "%s", &s); err != nil {
				return nil, err
			}
			if len(s) == 0 || s[0] != 'y' && s[0] != 'Y' {
				return nil, fmt.Errorf("user abort")
			}
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
			SigstorePrivateKey:  "",
			SigstorePassphrase:  nil,
		},
		PublicKey:               "",
		PrivateKey:              cc.privateKey,
		TLogUpload:              cc.tlogUpload,
		RecordCreationTimestamp: cc.recordCreationTimestamp,
		RekorURL:                cc.rekorURL,
		FulcioURL:               cc.fulcioURL,
		OIDCIssuer:              cc.oidcIssuer,
		OIDCClientID:            cc.oidcClientID,
		OIDCProvider:            cc.oidcProvider,
		InsecureSkipTLSVerify:   !cc.tlsVerify.Value(),
		AutoYes:                 cc.autoYes,
		SignManifestIndex:       cc.signManifestIndex,
		Registry:                cc.registry,
		Project:                 cc.project,
	})

	if err := s.InitGlobalSignerVerifier(signalContext); err != nil {
		return nil, fmt.Errorf("failed to init sign verifier: %w", err)
	}

	logrus.Infof("Arch List: [%v]", strings.Join(cc.arch, ","))
	logrus.Infof("OS List: [%v]", strings.Join(cc.os, ","))

	return s, err
}

// getRegistrySet only gets the registry set: map[registry-url]true.
func (cc *signCmd) getRegistrySet(images []string) map[string]bool {
	set := map[string]bool{}
	if cc.registry != "" {
		// The registry of image list were overrided by command option.
		set[cc.registry] = true
		return set
	}
	for _, line := range images {
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
			registry := utils.GetRegistryName(line)
			set[registry] = true
		default:
		}
	}
	return set
}
