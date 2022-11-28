package main

import (
	"flag"
	"fmt"
	"os"

	"cnrancher.io/image-tools/mirror"
	"github.com/sirupsen/logrus"
)

func init() {

}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(0)
	}

	// mirror reads file from image-list.txt or stdin and mirror image from
	// original repo to
	mirrorCmd := flag.NewFlagSet("mirror", flag.ExitOnError)
	mirrorFile := mirrorCmd.String("file", "", "the image list file")
	mirrorArch := mirrorCmd.String("arch", "x86_64,arm64", "the ARCH list of images, seperate with ','")
	mirrorSourceReg := mirrorCmd.String("source", "", "override the source registry")
	mirrorDestReg := mirrorCmd.String("dest", "", "override the destination registry")
	mirrorDebug := mirrorCmd.Bool("debug", false, "debug mode")

	switch os.Args[1] {
	case "mirror":
		mirrorCmd.Parse(os.Args[2:])
		if *mirrorDebug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		logrus.Debugln("mirrorFile: ", *mirrorFile)
		logrus.Debugln("mirrorArch: ", *mirrorArch)
		logrus.Debugln("sourceReg: ", *mirrorSourceReg)
		logrus.Debugln("destReg: ", *mirrorDestReg)
		mirror.MirrorImages(*mirrorFile, *mirrorArch, *mirrorSourceReg, *mirrorDestReg)
	case "":
	default:
		showHelp()
		os.Exit(0)
	}
}

func showHelp() {
	fmt.Printf("Usage: %s <sub-command> <parameters>\n", os.Args[0])

}
