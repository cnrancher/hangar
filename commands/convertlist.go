package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	LINE_FORMAT_UNKNOW = iota
	LINE_FORMAT_MIRROR // <SOURCE> <DEST> <TAG>
	LINE_FORMAT_SINGLE // registry.io/${REPOSITORY}/${NAME}:${TAG}
)

type convertListCmd struct {
	baseCmd
}

func newConvertListCmd() *convertListCmd {
	cc := &convertListCmd{}

	cc.baseCmd.cmd = &cobra.Command{
		Use:     "convert-list",
		Short:   "Convert images",
		Long:    `Convert images`,
		Example: `  hangar convert-list -i rancher-images.txt -o CONVERTED_MIRROR_LIST.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, config.DefaultProvider)

			if config.GetBool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
			}

			if err := cc.setupFlags(); err != nil {
				return err
			}
			if err := cc.run(); err != nil {
				return err
			}

			return nil
		},
	}

	cc.cmd.Flags().StringP("input", "i", "", "input image list (required)")
	cc.cmd.Flags().StringP("output", "o", "", "output image list (default \"[INPUT_FILE].converted\")")
	cc.cmd.Flags().StringP("source", "s", "", "specify the source registry")
	cc.cmd.Flags().StringP("destination", "d", "", "specify the destination registry")

	return cc
}

func (cc *convertListCmd) setupFlags() error {
	if config.GetString("input") == "" {
		return fmt.Errorf("input file not specified")
	}

	if config.GetString("output") == "" {
		config.Set("output", config.GetString("input")+".converted")
	}
	return nil
}

func (cc *convertListCmd) run() error {
	input := config.GetString("input")
	f, err := os.Open(input)
	if err != nil {
		logrus.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	convertedLines := []string{}
	for scanner.Scan() {
		l := scanner.Text()
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			continue
		}

		switch checkLineFormat(l) {
		case LINE_FORMAT_MIRROR:
			logrus.Info("input file is already 'mirror' format")
			return nil
		case LINE_FORMAT_SINGLE:
		default:
			// unknow format, continue
			logrus.Warnf("ignore line: %q: format unknow", l)
			continue
		}

		spec := make([]string, 0, 2)
		for _, v := range strings.Split(l, ":") {
			if len(v) > 0 {
				spec = append(spec, v)
			}
		}
		if len(spec) != 2 {
			if len(spec) == 1 {
				spec = append(spec, "latest")
			} else {
				logrus.Warnf("ignore line: %q: format unknow", l)
				continue
			}
		}

		var srcImage string
		source := config.GetString("source")
		if source == "" {
			srcImage = spec[0]
		} else {
			srcImage = utils.ConstructRegistry(spec[0], source)
		}
		dst := config.GetString("destination")
		destImage := utils.ConstructRegistry(spec[0], dst)
		outputLine := fmt.Sprintf("%s %s %s", srcImage, destImage, spec[1])
		// utils.AppendFileLine(cmdOutput, outputLine)
		convertedLines = append(convertedLines, outputLine)
	}
	output := config.GetString("output")
	utils.DeleteIfExist(output)
	utils.SaveSlice(output, convertedLines)
	logrus.Infof("Converted %q to %q", input, output)

	return nil
}

func checkLineFormat(line string) int {
	if isMirrorFormat(line) {
		return LINE_FORMAT_MIRROR
	} else if isSingleFormat(line) {
		return LINE_FORMAT_SINGLE
	}
	return 0
}

func isMirrorFormat(line string) bool {
	spec := make([]string, 0, 3)
	for _, v := range strings.Split(line, " ") {
		if len(v) > 0 {
			spec = append(spec, v)
		}
	}
	return len(spec) == 3
}

func isSingleFormat(line string) bool {
	if strings.Contains(line, " ") {
		return false
	}
	spec := make([]string, 0, 2)
	for _, v := range strings.Split(line, ":") {
		if len(v) > 0 {
			spec = append(spec, v)
		}
	}
	return len(spec) == 2 || len(spec) == 1
}
