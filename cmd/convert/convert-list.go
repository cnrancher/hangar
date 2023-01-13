// converter converts image list from 'single' format to 'mirror' format
package converter

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	u "github.com/cnrancher/image-tools/pkg/utils"
	"github.com/sirupsen/logrus"
)

const (
	LINE_FORMAT_UNKNOW = iota
	LINE_FORMAT_MIRROR // <SOURCE> <DEST> <TAG>
	LINE_FORMAT_SINGLE // registry.io/${REPOSITORY}/${NAME}:${TAG}
)

var (
	cmd          = flag.NewFlagSet("convert-list", flag.ExitOnError)
	cmdInput     = cmd.String("i", "", "input image list")
	cmdOutput    = cmd.String("o", "", "output image list")
	cmdSourceReg = cmd.String("s", "", "specify the source registry")
	cmdDestReg   = cmd.String("d", "", "specify the dest registry")
)

func Parse(args []string) {
	cmd.Parse(args)
}

func Convert() {
	if *cmdInput == "" {
		logrus.Error("Use '-i' to specify the input image list")
		cmd.Usage()
		os.Exit(1)
	}
	if *cmdOutput == "" {
		*cmdOutput = *cmdInput + ".converted"
	}

	f, err := os.Open(*cmdInput)
	if err != nil {
		logrus.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	u.DeleteIfExist(*cmdOutput)
	for scanner.Scan() {
		l := scanner.Text()
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			continue
		}

		switch checkLineFormat(l) {
		case LINE_FORMAT_MIRROR:
			logrus.Info("Input file is already 'mirror' format")
			return
		case LINE_FORMAT_SINGLE:
		default:
			// unknow format, continue
			logrus.Warnf("Unknown line format: %s", l)
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
				logrus.Warn("Unknow line: %s", l)
				continue
			}
		}

		var srcImage string
		if *cmdSourceReg == "" {
			srcImage = spec[0]
		} else {
			srcImage = u.ConstructRegistry(spec[0], *cmdSourceReg)
		}
		destImage := u.ConstructRegistry(spec[0], *cmdDestReg)
		outputLine := fmt.Sprintf("%s %s %s", srcImage, destImage, spec[1])
		u.AppendFileLine(*cmdOutput, outputLine)
	}
	logrus.Infof("Converted %q to %q", *cmdInput, *cmdOutput)
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
	spec := make([]string, 0, 2)
	for _, v := range strings.Split(line, ":") {
		if len(v) > 0 {
			spec = append(spec, v)
		}
	}
	return len(spec) == 2 || len(spec) == 1
}
