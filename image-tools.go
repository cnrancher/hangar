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

	// mirror subcmd reads file from image-list txt or stdin and mirror image
	// from source repo to the destination repo
	mirrorCmd := flag.NewFlagSet("mirror", flag.ExitOnError)
	mirrorFile := mirrorCmd.String("f", "", "image list file")
	mirrorArch := mirrorCmd.String("a", "x86_64,arm64", "architecture list of images, seperate with ','")
	mirrorSourceReg := mirrorCmd.String("s", "", "override the source registry")
	mirrorDestReg := mirrorCmd.String("d", "", "override the destination registry")
	// mirrorDestLoginURL := mirrorCmd.String("login-url", utils.DockerLoginURL, "destination registry login URL")
	mirrorDebug := mirrorCmd.Bool("debug", false, "enable the debug output")

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
		mirror.MirrorImages(
			*mirrorFile, *mirrorArch, *mirrorSourceReg,
			*mirrorDestReg)
	case "":
	default:
		showHelp()
		os.Exit(0)
	}
}

func showHelp() {
	fmt.Printf("Usage:\n\t%s <subcommand> <parameters>\n", os.Args[0])
	fmt.Printf("\t%s <subcommand> -h  -  get help info for subcommand\n", os.Args[0])
	fmt.Printf("\nSubcommand available: mirror\n")
}
