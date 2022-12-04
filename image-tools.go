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
	mirrorArch := mirrorCmd.String("a", "amd64,arm64", "architecture list of images, seperate with ','")
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
		logrus.Debugf("mirrorFile: %s", *mirrorFile)
		logrus.Debugf("mirrorArch: %s", *mirrorArch)
		logrus.Debugf("sourceReg: %s", *mirrorSourceReg)
		logrus.Debugf("destReg: %s", *mirrorDestReg)
		mirror.MirrorImages(*mirrorFile, *mirrorArch, *mirrorSourceReg, *mirrorDestReg)
	case "load": // TODO: load image from tar.gz tarball
	case "save": // TODO: save image to tar.gz tarball with image manifest
	default:
		showHelp()
		os.Exit(0)
	}
}

func showHelp() {
	fmt.Printf("Usage:\t%s COMMAND [OPTIONS]\n", os.Args[0])
	fmt.Println()
	fmt.Printf("Run '%s COMMAND --help' for more information on a command.\n", os.Args[0])
	fmt.Println()
	fmt.Printf("Commands: \n")
	fmt.Printf("  mirror \tMirror image from source registry to destination registry.\n")
	fmt.Printf("  load \t\tWIP.\n")
	fmt.Printf("  save \t\tWIP.\n")
}
