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

type signValidateOpts struct {
	file      string
	arch      []string
	os        []string
	publicKey string
	registry  string
	project   string
	failed    string
	jobs      int
	timeout   time.Duration
	tlsVerify commonFlag.OptionalBool

	exactRepository string
}

type signValidateCmd struct {
	*baseCmd
	*signValidateOpts
}

func newSignValidateCmd() *signValidateCmd {
	cc := &signValidateCmd{
		signValidateOpts: new(signValidateOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "validate -f IMAGE_LIST.txt",
		Short: "Validate the signed images with sigstore public key",
		Long: `Validate the signed images by sigstore public key with the
matchRepoDigestOrExact signedIdentity.`,
		Example: `# Validate the signed images by sigstore public key file.
hangar validate \
	--file IMAGE_LIST.txt \
	--sigstore-pubkey SIGSTORE.pub \
	--arch amd64,arm64 \
	--os linux \
	--exact-repository "registry.example.io/library/NAME"`,
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
			logrus.Infof("Validateing images in %q with sigstore public key %q",
				cc.file, cc.publicKey)
			if err := validate(h); err != nil {
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
	flags.StringVarP(&cc.publicKey, "sigstore-pubkey", "p", "", "sigstore public key file")
	flags.SetAnnotation("sigstore-pubkey", cobra.BashCompFilenameExt, []string{"pub"})
	flags.SetAnnotation("sigstore-pubkey", cobra.BashCompOneRequiredFlag, []string{""})
	flags.StringVarP(&cc.registry, "registry", "", "", "override all image registry URL in image list")
	flags.StringVarP(&cc.project, "project", "", "", "override all image projects in image list")
	flags.StringVarP(&cc.failed, "failed", "o", "sign-failed.txt", "file name of the sign failed image list")
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when validate each images")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")

	flags.StringVarP(&cc.exactRepository, "exact-repository", "", "",
		"validate the signed image with exactRepository signedIdentity")

	return cc
}

func (cc *signValidateCmd) prepareHangar() (hangar.Hangar, error) {
	if cc.file == "" {
		return nil, fmt.Errorf("image list file not provided, use '--file' option to specify the image list file")
	}
	if cc.publicKey == "" {
		return nil, fmt.Errorf("sigstore public key file not provided, " +
			"use '--sigstore-pubkey' option to specify the pub-key file")
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
	if _, err := os.Stat(cc.publicKey); err != nil {
		return nil, fmt.Errorf("failed to get status of file %q: %w",
			cc.publicKey, err)
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
			SigstorePublicKey:   cc.publicKey,
		},

		ExactRepository: cc.exactRepository,
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
