package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type archiveLsCmd struct {
	*baseCmd

	file string
	json bool
}

func newArchiveLsCmd() *archiveLsCmd {
	cc := &archiveLsCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "ls",
		Short: "Show images (index) in Hangar archive file",
		Long:  "",
		Example: `
# Show images in archive file:
hangar archive ls -f SAVED_ARCHIVE.zip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, cmdconfig.DefaultProvider)
			if cc.baseCmd.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("debug output enabled")
				logrus.Debugf("%v", utils.PrintObject(cmdconfig.Get("")))
			}

			if err := cc.run(); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.file, "file", "f", "", "Path to the Hangar archive file (.zip)")
	flags.SetAnnotation("file", cobra.BashCompFilenameExt, []string{"zip"})
	flags.SetAnnotation("file", cobra.BashCompOneRequiredFlag, []string{""})
	flags.BoolVarP(&cc.json, "json", "", false, "Output in json format")

	return cc
}

func (cc *archiveLsCmd) run() error {
	if cc.file == "" {
		return fmt.Errorf("file not provided, use '--file' to provide the Hangar archive file")
	}

	reader, err := archive.NewReader(cc.file)
	if err != nil {
		reader.Close()
		return fmt.Errorf("failed to open %q: %v", cc.file, err)
	}
	b, err := reader.Index()
	if err != nil {
		reader.Close()
		return fmt.Errorf("failed to get index from archive: %v", err)
	}
	reader.Close()

	index := archive.NewIndex()
	err = index.Unmarshal(b)
	if err != nil {
		return fmt.Errorf("failed to get index: %v", err)
	}

	if cc.json {
		b, _ := json.MarshalIndent(index, "", "  ")
		fmt.Print(string(b))
		return nil
	}
	logrus.Infof("Created time: %v", index.Time)
	logrus.Infof("Index version: %v", index.Version)
	logrus.Infof("Images:")
	for i, image := range index.List {
		fmt.Printf("%4d | %s:%s | %s | %s\n",
			i+1, image.Source, image.Tag,
			strings.Join(image.ArchList, ","),
			strings.Join(image.OsList, ","))
	}
	return nil
}
