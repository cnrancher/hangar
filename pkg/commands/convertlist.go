package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type convertListCmd struct {
	*baseCmd

	input       string
	output      string
	source      string
	destination string
	autoYes     bool
}

func newConvertListCmd() *convertListCmd {
	cc := &convertListCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "convert-list -i IMAGE_LIST.txt -o OUTPUT_IMAGE_LIST.txt",
		Short: "Convert the image list format to mirror format (see example of this command).",
		Example: `
# Prepare an image list file with default format:
docker.io/library/mysql:8
docker.io/library/nginx:latest

# Use following command to convert the image list format to 'mirror' format.
hangar convert-list \
  	--input IMAGE_LIST.txt \
	--output OUTPUT_IMAGE_LIST.txt \
	--source docker.io \
	--destination registry.example.io

# The converted image list is:
docker.io/library/mysql registry.example.io/library/mysql 8
docker.io/library/nginx registry.example.io/library/nginx latest`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, cmdconfig.DefaultProvider)
			if err := cc.setupFlags(); err != nil {
				return err
			}
			if err := cc.run(); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.cmd.Flags()
	flags.StringVarP(&cc.input, "input", "i", "", "input image list file")
	flags.SetAnnotation("input", cobra.BashCompFilenameExt, []string{"txt"})
	flags.SetAnnotation("input", cobra.BashCompOneRequiredFlag, []string{""})
	flags.StringVarP(&cc.output, "output", "o", "", "output image list (default \"[INPUT_FILE].converted\")")
	flags.SetAnnotation("output", cobra.BashCompFilenameExt, []string{"txt"})
	flags.StringVarP(&cc.source, "source", "s", "", "specify the source registry (optional)")
	flags.StringVarP(&cc.destination, "destination", "d", "", "specify the destination registry (optional)")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")
	return cc
}

func (cc *convertListCmd) setupFlags() error {
	if cc.baseCmd.debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debugf("debug output enabled")
		logrus.Debugf("%v", utils.PrintObject(cmdconfig.Get("")))
	}
	if cc.input == "" {
		return fmt.Errorf("input file not specified")
	}
	if cc.output == "" {
		cc.output = cc.input + ".converted"
	}
	return nil
}

func (cc *convertListCmd) run() error {
	f, err := os.Open(cc.input)
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

		switch imagelist.Detect(l) {
		case imagelist.TypeMirror:
			logrus.Infof("Skip line: %v", l)
			continue
		case imagelist.TypeDefault:
		default:
			// unknow format, continue
			logrus.Warnf("Ignore line: %q: format unknow", l)
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
				logrus.Warnf("Ignore line: %q: format unknow", l)
				continue
			}
		}

		var srcImage string
		if cc.source == "" {
			srcImage = spec[0]
		} else {
			srcImage = utils.ConstructRegistry(spec[0], cc.source)
		}
		dst := cc.destination
		destImage := utils.ConstructRegistry(spec[0], dst)
		outputLine := fmt.Sprintf("%s %s %s", srcImage, destImage, spec[1])
		logrus.Debugf("converted %q => %q", l, outputLine)
		convertedLines = append(convertedLines, outputLine)
	}

	if err := utils.CheckFileExistsPrompt(signalContext, cc.output, cc.autoYes); err != nil {
		return err
	}

	file, err := os.OpenFile(cc.output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to save %q: %v", cc.output, err)
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "%v", strings.Join(convertedLines, "\n"))
	if err != nil {
		return fmt.Errorf("failed to write %q: %v", cc.output, err)
	}
	logrus.Infof("Converted %q to %q", cc.input, cc.output)

	return nil
}
