package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type archiveMergeOpts struct {
	files   []string
	output  string
	autoYes bool
}

type archiveMergeCmd struct {
	*baseCmd
	*archiveMergeOpts
}

func newArchiveMergeCmd() *archiveMergeCmd {
	cc := &archiveMergeCmd{
		archiveMergeOpts: &archiveMergeOpts{},
	}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "merge",
		Aliases: []string{"m"},
		Short:   "Merge multiple hangar archive files into one new archive file",
		Long:    "",
		Example: `
# Merge multiple archive files
hangar archive merge \
	--file ARCHIVE_1.zip \
	--file ARCHIVE_2.zip \
	--output MERGE_OUTPUT.zip`,
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

	flags := cc.baseCmd.cmd.Flags()
	flags.StringSliceVarP(&cc.files, "file", "f", nil, "archive file path")
	flags.StringVarP(&cc.output, "output", "o", "", "output archive file")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")

	return cc
}

func (cc *archiveMergeCmd) run() error {
	if len(cc.files) == 0 {
		return fmt.Errorf("archive files not provided, use '--file' option to specify at least 2 archive files")
	}
	if len(cc.files) < 2 {
		return fmt.Errorf("use '--file' option to specify at least 2 archive files")
	}
	if cc.output == "" {
		return fmt.Errorf("output file not provided, use '--output' to specify the output file")
	}
	if err := cc.merge(); err != nil {
		return err
	}

	return nil
}

func (cc *archiveMergeCmd) merge() error {
	var (
		readers  = []*archive.Reader{}
		writer   *archive.Writer
		err      error
		newIndex = archive.NewIndex()
	)
	for _, fn := range cc.files {
		if _, err := os.Stat(fn); err != nil {
			return fmt.Errorf("stat %q: %w", fn, err)
		}
		ar, err := archive.NewReader(fn)
		if err != nil {
			return fmt.Errorf("failed to create reader for %q: %w", fn, err)
		}
		readers = append(readers, ar)
		defer ar.Close()
	}
	if len(readers) == 0 {
		return fmt.Errorf("no archive files to merge")
	}
	if err := utils.CheckFileExistsPrompt(signalContext, cc.output, cc.autoYes); err != nil {
		return err
	}
	writer, err = archive.NewWriter(cc.output)
	if err != nil {
		return fmt.Errorf("failed to create writer for %q: %w", cc.output, err)
	}
	defer writer.Close()

	for _, reader := range readers {
		ib, err := reader.Index()
		if err != nil {
			return fmt.Errorf("failed to load index: %w", err)
		}
		i := archive.NewIndex()
		if err := i.Unmarshal(ib); err != nil {
			return fmt.Errorf("failed to read index data: %w", err)
		}

		for _, img := range i.List {
			if err := writer.CopyImage(img, reader); err != nil {
				return fmt.Errorf("failed to copy image [%v:%v]: %w", img.Source, img.Tag, err)
			}
			newIndex.Append(img)
			logrus.Infof("Copy [%s:%s]", img.Source, img.Tag)
		}
	}

	if err := writer.WriteIndex(newIndex); err != nil {
		return fmt.Errorf("failed to write index to %q: %w", cc.output, err)
	}
	logrus.Infof("Merged archive files [%v] to %q", strings.Join(cc.files, ","), cc.output)

	return nil
}
