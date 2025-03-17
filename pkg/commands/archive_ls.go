package commands

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type archiveLsCmd struct {
	*baseCmd

	file      string
	json      bool
	imageOnly bool
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
	flags.StringVarP(&cc.file, "file", "f", "", "Path to the Hangar archive file (.zip)")
	flags.SetAnnotation("file", cobra.BashCompFilenameExt, []string{"zip"})
	flags.SetAnnotation("file", cobra.BashCompOneRequiredFlag, []string{""})
	flags.BoolVarP(&cc.json, "json", "", false, "Output in json format")
	flags.BoolVarP(&cc.imageOnly, "image-only", "", false, "Only output image list")

	return cc
}

func (cc *archiveLsCmd) run(args []string) error {
	if cc.file == "" {
		if len(args) > 0 {
			cc.file = args[0]
		} else {
			return fmt.Errorf("file not provided, use '--file' to provide the Hangar archive file")
		}
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
	defer reader.Close()

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
	if cc.imageOnly {
		for _, image := range index.List {
			fmt.Printf("%s:%s\n", image.Source, image.Tag)
		}
		return nil
	}
	if !cc.baseCmd.hideLogTime {
		logrus.Infof("Created time: %v", index.Time.Format(time.DateOnly))
	}
	logrus.Infof("Index version: %v", index.Version)
	logrus.Infof("Images:")

	const unknownPlatform = "unknown"
	for i, image := range index.List {
		layers := map[string]bool{}
		for _, img := range image.Images {
			for _, l := range img.Layers {
				layers[l.Hex()] = true
			}
		}
		size := 0.0
		for l := range layers {
			n := fmt.Sprintf("share/sha256/%v", l)
			s, err := reader.FileCompressedSize(n)
			if err != nil {
				logrus.Warnf("failed to get %v layer compressed size: %v", l, err)
			}
			size += float64(s)
		}

		hasProvenance := false
		if i := slices.Index(image.ArchList, unknownPlatform); i >= 0 {
			hasProvenance = true
			// image.ArchList = append(image.ArchList[:i], image.ArchList[i+1:]...)
			image.ArchList = slices.Delete(image.ArchList, i, i+1)
		}
		if i := slices.Index(image.OsList, unknownPlatform); i >= 0 {
			hasProvenance = true
			// image.OsList = append(image.OsList[:i], image.OsList[i+1:]...)
			image.OsList = slices.Delete(image.OsList, i, i+1)
		}
		var s string
		archList := strings.Join(image.ArchList, ",")
		osList := strings.Join(image.OsList, ",")
		if archList == "" {
			archList = "NOARCH"
		}
		if osList == "" {
			osList = "NOOS"
		}
		switch {
		case size < 10e5:
			s = fmt.Sprintf("%4d | %s:%s | %s | %s | %.2fK",
				i+1, image.Source, image.Tag,
				archList,
				osList,
				size/1024.0)
		case size < 10e9:
			s = fmt.Sprintf("%4d | %s:%s | %s | %s | %.2fM",
				i+1, image.Source, image.Tag,
				archList,
				osList,
				size/1024.0/1024.0)
		case size < 10e12:
			s = fmt.Sprintf("%4d | %s:%s | %s | %s | %.2fG",
				i+1, image.Source, image.Tag,
				archList,
				osList,
				size/1024.0/1024.0/1024.0)
		}

		if hasProvenance {
			s += " | (with attestation)"
		}
		fmt.Println(s)
	}
	return nil
}
