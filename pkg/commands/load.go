package commands

import (
	"bufio"
	"bytes"
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

type loadOpts struct {
	file           string
	arch           []string
	os             []string
	source         string
	sourceRegistry string
	destination    string
	failed         string
	jobs           int
	timeout        time.Duration
	project        string
	skipLogin      bool
	copyProvenance bool
	overwriteExist bool
	tlsVerify      commonFlag.OptionalBool

	sigstorePrivateKey     string
	sigstorePassphraseFile string
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
		Use:     "load -s SAVED_ARCHIVE.zip -d REGISTRY_SERVER",
		Aliases: []string{"l"},
		Short:   "Load images from zip archive created by 'save' command to registry server",
		Long: `Load images from zip archive created by 'save' command to registry server.

The load command will create Harbor V2 projects for destination registry automatically.
`,
		Example: `# Load images from SAVED_ARCHIVE.zip to REGISTRY server
# and sign the loaded images by sigstore private key file.
hangar load \
	--file IMAGE_LIST.txt \
	--source SAVED_ARCHIVE.zip \
	--destination REGISTRY_URL \
	--arch amd64,arm64 \
	--os linux \
	--sigstore-private-key SIGSTORE.key`,
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
	flags.IntVarP(&cc.jobs, "jobs", "j", 1, "worker number, copy images parallelly (1-20)")
	flags.DurationVarP(&cc.timeout, "timeout", "", time.Minute*10, "timeout when save each images")
	flags.StringVarP(&cc.project, "project", "", "", "override all destination image projects")
	flags.BoolVarP(&cc.copyProvenance, "provenance", "", true, "copy SLSA provenance")
	flags.BoolVarP(&cc.overwriteExist, "overwrite", "", false,
		"overwrite exist manifest index in destination registry")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")

	flags.BoolVarP(&cc.skipLogin, "skip-login", "", false,
		"skip check the destination registry is logged in (used in shell script)")

	flags.StringVarP(&cc.sigstorePrivateKey, "sigstore-private-key", "", "",
		"sign images by sigstore private key when mirroring")
	flags.MarkDeprecated("sigstore-private-key", "signv1 is deprecated, use 'hangar sign' instead")
	flags.MarkHidden("sigstore-private-key")
	flags.StringVarP(&cc.sigstorePassphraseFile, "sigstore-passphrase-file", "", "",
		"passphrase file of the sigstore private key")
	flags.MarkDeprecated("sigstore-passphrase-file", "signv1 is deprecated, use 'hangar sign' instead")
	flags.MarkHidden("sigstore-passphrase-file")

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
		logrus.Debugf("Debug mode enabled, force worker number to 1")
		cc.jobs = 1
	} else {
		if cc.jobs > utils.MaxWorkerNum || cc.jobs < utils.MinWorkerNum {
			logrus.Warnf("invalid worker num: %v, set to 1", cc.jobs)
			cc.jobs = 1
		}
	}

	var images []string
	if cc.file != "" {
		file, err := os.Open(cc.file)
		if err != nil {
			return nil, fmt.Errorf("failed to open %q: %w", cc.file, err)
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
			return nil, fmt.Errorf("failed to close %q: %w", cc.file, err)
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

	var passphrase []byte
	if cc.sigstorePrivateKey != "" {
		logrus.Warnf("DEPRECATED: signv1 is deprecated, use 'hangar sign' to sign image instead!")
	}
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

	l, err := hangar.NewLoader(&hangar.LoaderOpts{
		CommonOpts: hangar.CommonOpts{
			Images:              images,
			Arch:                cc.arch,
			OS:                  cc.os,
			Variant:             nil,
			CopyProvenance:      cc.copyProvenance,
			OverwriteExist:      cc.overwriteExist,
			Timeout:             cc.timeout,
			Workers:             cc.jobs,
			FailedImageListName: cc.failed,
			SystemContext:       sysCtx,
			Policy:              policy,
			SigstorePrivateKey:  cc.sigstorePrivateKey,
			SigstorePassphrase:  passphrase,
		},

		SourceRegistry:      cc.sourceRegistry,
		DestinationRegistry: cc.destination,
		DestinationProject:  cc.project,
		SharedBlobDirPath:   "", // Use the default shared blob dir path.
		ArchiveName:         cc.source,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create loader: %w", err)
	}
	logrus.Infof("Arch List: [%v]", strings.Join(cc.arch, ","))
	logrus.Infof("OS List: [%v]", strings.Join(cc.os, ","))

	return l, nil
}
