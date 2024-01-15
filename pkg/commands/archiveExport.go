package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type archiveExportOpts struct {
	file           string
	source         string
	sourceRegistry string
	destination    string
	failed         string
	autoYes        bool
}

type archiveExportCmd struct {
	*baseCmd
	*archiveExportOpts
}

func newArchiveExportCmd() *archiveExportCmd {
	cc := &archiveExportCmd{
		archiveExportOpts: &archiveExportOpts{},
	}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "export",
		Short: "Export images from hangar archive file into a new archive file",
		Long:  "Export some images from hangar archive file into a new archive file by image list file.",
		Example: `
# Export images from archive file
hangar archive export \
	--file IMAGE_LIST.txt \
	--source SAVED_ARCHIVE.zip \
	--destination EXPORT_OUTPUT.zip`,
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
	flags.StringVarP(&cc.file, "file", "f", "", "image list file (required)")
	flags.StringVarP(&cc.source, "source", "s", "", "source archive file")
	flags.StringVarP(&cc.sourceRegistry, "source-registry", "", "", "override the source registry of image list file")
	flags.StringVarP(&cc.destination, "destination", "d", "", "destination archive file")
	flags.StringVarP(&cc.failed, "failed", "", "export-failed.txt", "export failed image list file name")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")

	return cc
}

func (cc *archiveExportCmd) run() error {
	if cc.file == "" {
		return fmt.Errorf("image list file not provided, use '--file' to specify the image list file")
	}
	if cc.source == "" {
		return fmt.Errorf("source archive file not provided, use '--source' to specify the source archive")
	}
	if cc.destination == "" {
		return fmt.Errorf("destination archive file not provided, use '--destination' to specify the output archive file")
	}
	if err := utils.CheckFileExistsPrompt(signalContext, cc.destination, cc.autoYes); err != nil {
		return err
	}
	if err := cc.export(); err != nil {
		return err
	}

	return nil
}

func (cc *archiveExportCmd) export() error {
	var images []string
	file, err := os.Open(cc.file)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", cc.file, err)
	}
	sc := bufio.NewScanner(file)
	sc.Split(bufio.ScanLines)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			continue
		}
		switch imagelist.Detect(l) {
		case imagelist.TypeDefault:
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", l)
			continue
		}
		images = append(images, l)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close %q: %w", cc.file, err)
	}

	ar, err := archive.NewReader(cc.source)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer ar.Close()

	aw, err := archive.NewWriter(cc.destination)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer aw.Close()

	ib, err := ar.Index()
	if err != nil {
		return fmt.Errorf("failed to read index from archive: %w", err)
	}
	index := archive.NewIndex()
	index.Unmarshal(ib)
	newIndex := archive.NewIndex()
	failed := make([]string, 0)

	for _, line := range images {
		registry := utils.GetRegistryName(line)
		if cc.sourceRegistry != "" {
			registry = cc.sourceRegistry
		}
		project := utils.GetProjectName(line)
		name := utils.GetImageName(line)
		tag := utils.GetImageTag(line)
		imageSource := fmt.Sprintf("%s/%s/%s", registry, project, name)
		var image *archive.Image
		for _, i := range index.List {
			if !(i.Source == imageSource && i.Tag == tag) {
				continue
			}
			image = i
		}
		if image == nil {
			logrus.Errorf("Failed to export image [%v]: image not found in archive file", line)
			failed = append(failed, line)
			continue
		}
		err := aw.CopyImage(image, ar)
		if err != nil {
			logrus.Warnf("Error occured when copy image [%v:%v]", image.Source, image.Tag)
			return err
		}
		newIndex.Append(image)
		logrus.Infof("Copy [%s:%s]", image.Source, image.Tag)
	}
	if err := aw.WriteIndex(newIndex); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}
	if len(failed) > 0 {
		f, err := os.OpenFile(cc.failed, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to create %q: %w", cc.failed, err)
		}
		_, err = f.WriteString(strings.Join(failed, "\n"))
		if err != nil {
			return fmt.Errorf("failed to write %q: %w", cc.failed, err)
		}
		logrus.Errorf("Export failed image list: \n%v", strings.Join(failed, "\n"))
		logrus.Errorf("Export failed images saved to %q", cc.failed)
		return fmt.Errorf("some images failed to export")
	}

	return nil
}
