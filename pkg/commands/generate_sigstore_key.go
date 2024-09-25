package commands

import (
	"bytes"
	"fmt"
	"os"
	"slices"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/signature/sigstore"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type generateSigstoreKeyOpts struct {
	outputPrefix   string
	passphraseFile string
	autoYes        bool
}

type generateSigstoreKeyCmd struct {
	*baseCmd
	*generateSigstoreKeyOpts
}

func newgenerateSigstoreKeyCmd() *generateSigstoreKeyCmd {
	cc := &generateSigstoreKeyCmd{
		generateSigstoreKeyOpts: new(generateSigstoreKeyOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "generate-sigstore-key -p NAME",
		Short:   "Generate a sigstore key-pair for signing images",
		Long:    ``,
		Example: `hangar generate-sigstore-key --prefix sigstore`,
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cc.run(); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.baseCmd.cmd.PersistentFlags()
	flags.StringVarP(&cc.outputPrefix, "prefix", "p", "sigstore",
		"prefix name for the generated sigstore '.pub' and '.key' files")
	flags.StringVar(&cc.passphraseFile, "passphrase-file", "",
		"read the passphrase for the private key from file")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false,
		"answer yes automatically (used in shell script)")

	return cc
}

func (cc *generateSigstoreKeyCmd) run() error {
	if cc.outputPrefix == "" {
		logrus.Errorf("Sigstore key-pair name prefix not provided")
		return fmt.Errorf("usage: generate-key --prefix PREFIX")
	}

	publicKeyPath := cc.outputPrefix + ".pub"
	privateKeyPath := cc.outputPrefix + ".key"

	if err := utils.CheckFileExistsPrompt(signalContext, publicKeyPath, cc.autoYes); err != nil {
		return err
	}
	if err := utils.CheckFileExistsPrompt(signalContext, privateKeyPath, cc.autoYes); err != nil {
		return err
	}

	var passphrase []byte
	if cc.passphraseFile != "" {
		b, err := os.ReadFile(cc.passphraseFile)
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", cc.passphraseFile, err)
		}
		b = bytes.TrimSpace(b)
		logrus.Infof("Read the passphrase for key %q from %q",
			privateKeyPath, cc.passphraseFile)
		passphrase = b
	} else {
		fmt.Printf("Enter the passphrase for key %q: ", privateKeyPath)
		p1, err := utils.ReadPassword(signalContext)
		if err != nil {
			return err
		}
		fmt.Printf("Enter the passphrase again: ")
		p2, err := utils.ReadPassword(signalContext)
		if err != nil {
			return err
		}
		if !slices.Equal(p1, p2) {
			return fmt.Errorf("password does not match")
		}
		passphrase = p1
	}
	if len(passphrase) < 5 {
		logrus.Warnf("The passphrase of key %q is too weak!", privateKeyPath)
	}
	keys, err := sigstore.GenerateKeyPair(passphrase)
	if err != nil {
		return fmt.Errorf("failed to generating key pair: %w", err)
	}
	if err := os.WriteFile(privateKeyPath, keys.PrivateKey, 0600); err != nil {
		return fmt.Errorf("failed to write private key to %q: %w", privateKeyPath, err)
	}
	if err := os.WriteFile(publicKeyPath, keys.PublicKey, 0644); err != nil {
		return fmt.Errorf("failed to write public key to %q: %w", publicKeyPath, err)
	}
	logrus.Infof("Write sigstore key-pair to %q, %q", publicKeyPath, privateKeyPath)
	return nil
}
