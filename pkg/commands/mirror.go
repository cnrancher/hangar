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

type mirrorOpts struct {
	file           string
	arch           []string
	os             []string
	source         string
	destination    string
	failed         string
	jobs           int
	timeout        time.Duration
	skipLogin      bool
	copyProvenance bool
	tlsVerify      commonFlag.OptionalBool

	sourceProject      string
	destinationProject string

	removeSignatures       bool
	sigstorePrivateKey     string
	sigstorePassphraseFile string
}

type mirrorCmd struct {
	*baseCmd
	*mirrorOpts
}

func newMirrorCmd() *mirrorCmd {
	cc := &mirrorCmd{
		mirrorOpts: new(mirrorOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "mirror -f IMAGE_LIST.txt -d DESTINATION_REGISTRY",
		Short: "Mirror images between registry servers",
		Long:  ``,
		Example: `# Mirror images from SOURCE REGISTRY to DESTINATION REGISTRY.
hangar mirror \
	--file IMAGE_LIST.txt \
	--source SOURCE_REGISTRY \
	--destination DESTINATION_REGISTRY \
	--arch amd64,arm64 \
	--os linux \
	--sigstore-private-key SIGSTORE.key`,
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
	flags.StringVarP(&cc.destination, "destination", "d", "", "specify the destination image registry")
	flags.StringVarP(&cc.failed, "failed", "o", "mirror-failed.txt", "file name of the mirror failed image list")
	flags.SetAnnotation("failed", cobra.BashCompFilenameExt, []string{"txt"})
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when mirror each images")
	flags.BoolVarP(&cc.copyProvenance, "provenance", "", true, "copy SLSA provenance")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")

	flags.BoolVarP(&cc.skipLogin, "skip-login", "", false,
		"skip check the destination registry is logged in (used in shell script)")
	flags.StringVarP(&cc.sourceProject, "source-project", "", "",
		"override all source image projects")
	flags.StringVarP(&cc.destinationProject, "destination-project", "", "",
		"override all destination image projects")
	flags.BoolVarP(&cc.removeSignatures, "remove-signatures", "", false,
		"do not copy image signatures when mirror images")
	flags.StringVarP(&cc.sigstorePrivateKey, "sigstore-private-key", "", "",
		"sign images by sigstore private key when mirror images")
	flags.StringVarP(&cc.sigstorePassphraseFile, "sigstore-passphrase-file", "", "",
		"passphrase file of the sigstore private key")

	addCommands(
		cc.cmd,
		newMirrorValidateCmd(cc.mirrorOpts),
	)

	return cc
}

func (cc *mirrorCmd) prepareHangar() (hangar.Hangar, error) {
	if cc.file == "" {
		return nil, fmt.Errorf("file not provided")
	}
	// if cc.destination == "" {
	// 	return fmt.Errorf("destination registry URL not provided")
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
	if cc.sigstorePrivateKey != "" && cc.sigstorePassphraseFile != "" {
		b, err := os.ReadFile(cc.sigstorePassphraseFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read %q: %w",
				cc.sigstorePassphraseFile, err)
		}
		b = bytes.TrimSpace(b)
		logrus.Infof("Read the passphrase for key %q from %q",
			cc.sigstorePrivateKey, cc.sigstorePassphraseFile)
		passphrase = b
	} else if cc.sigstorePrivateKey != "" {
		var err error
		fmt.Printf("Enter the passphrase for key %q: ", cc.sigstorePrivateKey)
		passphrase, err = utils.ReadPassword(signalContext)
		if err != nil {
			return nil, err
		}
	}

	m, err := hangar.NewMirrorer(&hangar.MirrorerOpts{
		CommonOpts: hangar.CommonOpts{
			Images:              images,
			Arch:                cc.arch,
			OS:                  cc.os,
			Variant:             nil, // TODO: support variants
			CopyProvenance:      cc.copyProvenance,
			Timeout:             cc.timeout,
			Workers:             cc.jobs,
			FailedImageListName: cc.failed,
			SystemContext:       sysCtx,
			Policy:              policy,
			RemoveSignatures:    cc.removeSignatures,
			SigstorePrivateKey:  cc.sigstorePrivateKey,
			SigstorePassphrase:  passphrase,
		},

		SourceRegistry:      cc.source,
		SourceProject:       cc.sourceProject,
		DestinationRegistry: cc.destination,
		DestinationProject:  cc.destinationProject,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mirrorer: %v", err)
	}
	logrus.Infof("Arch List: [%v]", strings.Join(cc.arch, ","))
	logrus.Infof("OS List: [%v]", strings.Join(cc.os, ","))

	return m, nil
}

// getRegistrySet only gets the destination registry set: map[registry-url]true.
func (cc *mirrorCmd) getRegistrySet(images []string) map[string]bool {
	set := map[string]bool{}
	if cc.destination != "" {
		// The registry of image list were overrided by command option.
		set[cc.destination] = true
		return set
	}
	for _, line := range images {
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
			registry := utils.GetRegistryName(line)
			set[registry] = true
		case imagelist.TypeMirror:
			spec, _ := imagelist.GetMirrorSpec(line)
			if len(spec) != 3 {
				continue
			}
			set[utils.GetRegistryName(spec[1])] = true
		default:
		}
	}
	return set
}
