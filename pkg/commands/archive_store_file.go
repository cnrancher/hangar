package commands

import (
	"fmt"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/hangar/archive/oci"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type storeFileOpts struct {
	file      string
	tlsVerify bool
}

type storeFile struct {
	*baseCmd
	*storeFileOpts
}

func newStoreFileCmd() *storeFile {
	cc := &storeFile{
		storeFileOpts: new(storeFileOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "file",
		Short:   "Store OCI format Custom File in archive",
		Aliases: []string{"files", "f"},
		Long:    "",
		Example: `# Add files to archive
hangar archive store file \
	--file saved_images.zip \
	./path/to/file1.txt \
	./path/to/file2.txt \

# Add file from URL
hangar archive store file \
	--file saved_images.zip \
	https://example.com/path/to/file.txt`,
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cc.run(args); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.file, "file", "f", "", "Path to the Hangar archive file (zip)")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "Require HTTPS and verify certificates")

	return cc
}

func (cc *storeFile) run(args []string) error {
	if len(args) == 0 {
		cc.cmd.Help()
		return fmt.Errorf("file not provided")
	}
	if cc.file == "" {
		return fmt.Errorf("archive file not provided")
	}

	policy, err := cc.getPolicy()
	if err != nil {
		return fmt.Errorf("failed to get policy: %w", err)
	}
	au, err := archive.NewUpdater(cc.file)
	if err != nil {
		return err
	}
	defer au.Close()

	for _, a := range args {
		file := oci.NewFile(&oci.FileOptions{
			CommonOpts: oci.CommonOpts{
				InsecureSkipVerify: !cc.tlsVerify,
				SystemContext:      cc.baseCmd.newSystemContext(),
				Policy:             policy,
			},
			URL: a,
		})
		if err := file.Fetch(signalContext); err != nil {
			return fmt.Errorf("failed to add %q: %w",
				a, err)
		}
		if err := file.WriteArchive(au); err != nil {
			return fmt.Errorf("failed to write OCI image path %q to archive: %w",
				file.CacheDir(), err)
		}
		logrus.Infof("Add OCI File [%v]", file.Source())
	}

	return nil
}
