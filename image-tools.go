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
	mirrorFile := mirrorCmd.String("f", "", "the image list file")
	mirrorArch := mirrorCmd.String("a", "x86_64,arm64", "the ARCH list of images, seperate with ','")
	mirrorSourceReg := mirrorCmd.String("s", "", "override the source registry")
	mirrorDestReg := mirrorCmd.String("d", "", "override the destination registry")
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
	fmt.Printf("Usage: %s <subcommand> <parameters>\n", os.Args[0])
	fmt.Printf("Run '%s <subcommand> -h' for more info.\n", os.Args[0])
	fmt.Printf("Subcommands: mirror\n")
}
