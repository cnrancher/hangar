package commands

import (
	"bufio"
	"bytes"
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

type signOpts struct {
	file           string
	arch           []string
	os             []string
	privateKey     string
	passphraseFile string
	registry       string
	project        string
	failed         string
	jobs           int
	timeout        time.Duration
	skipLogin      bool
	tlsVerify      commonFlag.OptionalBool
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
		Use:   "sign -f IMAGE_LIST.txt --key SIGSTORE.key",
		Short: "Sign multiple container images with sigstore private key",
		Long:  ``,
		Example: `# Sign the images with sigstore private key file.
hangar sign \
	--file IMAGE_LIST.txt \
	--sigstore-key SIGSTORE.key \
	--sigstore-passphrase-file "/path/to/passphrase/file" \
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
	flags.StringVarP(&cc.privateKey, "sigstore-key", "k", "", "sigstore private key file")
	flags.SetAnnotation("sigstore-key", cobra.BashCompFilenameExt, []string{"key", "private"})
	flags.SetAnnotation("sigstore-key", cobra.BashCompOneRequiredFlag, []string{""})
	flags.StringVar(&cc.passphraseFile, "sigstore-passphrase-file", "",
		"read the passphrase for the private key from file")
	flags.StringVarP(&cc.registry, "registry", "", "", "override all image registry URL in image list")
	flags.StringVarP(&cc.project, "project", "", "", "override all image projects in image list")
	flags.StringVarP(&cc.failed, "failed", "o", "sign-failed.txt", "file name of the sign failed image list")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, sign images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when sign each images")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")

	flags.BoolVarP(&cc.skipLogin, "skip-login", "", false,
		"skip check the registry is logged in (used in shell script)")

	addCommands(
		cc.cmd,
		newSignValidateCmd(),
	)

	return cc
}

func (cc *signCmd) prepareHangar() (hangar.Hangar, error) {
	if cc.file == "" {
		return nil, fmt.Errorf("image list file not provided, use '--file' option to specify the image list file")
	}
	if cc.privateKey == "" {
		return nil, fmt.Errorf("sigstore private key file not provided, " +
			"use '--sigstore-key' option to specify the key file")
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

	policy, err := cc.getPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	var passphrase []byte
	if cc.passphraseFile != "" {
		b, err := os.ReadFile(cc.passphraseFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read %q: %w", cc.passphraseFile, err)
		}
		b = bytes.TrimSpace(b)
		logrus.Infof("Read the passphrase for key %q from %q",
			cc.privateKey, cc.passphraseFile)
		passphrase = b
	} else {
		var err error
		fmt.Printf("Enter the passphrase for key %q: ", cc.privateKey)
		passphrase, err = utils.ReadPassword(signalContext)
		if err != nil {
			return nil, err
		}
	}

	s, err := hangar.NewSigner(&hangar.SignerOpts{
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
			SigstorePrivateKey:  cc.privateKey,
			SigstorePassphrase:  passphrase,
		},

		ExactRepository: "", // ExactRepository is only used for verifying.
		Registry:        cc.registry,
		Project:         cc.project,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %v", err)
	}
	logrus.Infof("Arch List: [%v]", strings.Join(cc.arch, ","))
	logrus.Infof("OS List: [%v]", strings.Join(cc.os, ","))

	return s, nil
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
