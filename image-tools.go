package main

import (
	"flag"
	"fmt"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

var (
	dockerUsername = os.Getenv("DOCKER_USERNAME")
	dockerPassword = os.Getenv("DOCKER_PASSWORD")
	dockerRegistry = os.Getenv("DOCKER_REGISTRY")
)

// mirror COMMAND reads file from image-list txt or stdin, then mirror images
// from source repo to the destination repo
var (
	mirrorCmd       = flag.NewFlagSet("mirror", flag.ExitOnError)
	mirrorFile      = mirrorCmd.String("f", "", "image list file")
	mirrorArch      = mirrorCmd.String("a", "amd64,arm64", "architecture list of images, seperate with ','")
	mirrorSourceReg = mirrorCmd.String("s", "", "override the source registry")
	mirrorDestReg   = mirrorCmd.String("d", "", "override the destination registry")
	mirrorFailed    = mirrorCmd.String("o", "mirror-failed.txt", "file name of the mirror failed image list")
	mirrorDebug     = mirrorCmd.Bool("debug", false, "enable the debug output")
	mirrorJobs      = mirrorCmd.Int("j", 1, "job number, async mode if larger than 1, maximun is 20")
)

var (
	saveCmd       = flag.NewFlagSet("save", flag.ExitOnError)
	saveFile      = saveCmd.String("f", "", "image list file")
	saveArch      = saveCmd.String("a", "amd64,arm64", "architecture list of images, seperate with ','")
	saveSourceReg = saveCmd.String("s", "", "override the source registry")
	saveDestDir   = saveCmd.String("d", "./output/", "specify the output directory")
	saveFailed    = saveCmd.String("o", "save-failed.txt", "file name of the save failed image list")
	saveDebug     = saveCmd.Bool("debug", false, "enable the debug output")
	saveJobs      = saveCmd.Int("j", 1, "job number, async mode if larger than 1, maximum is 20")
)

var (
	loadCmd     = flag.NewFlagSet("load", flag.ExitOnError)
	loadFile    = loadCmd.String("f", "", "saved tar.gz file")
	loadDestReg = loadCmd.String("d", "", "override the destination registry")
	loadFailed  = loadCmd.String("o", "load-failed.txt", "file name of the load failed image list")
	loadDebug   = loadCmd.Bool("debug", false, "enable the debug output")
	loadJobs    = loadCmd.Int("j", 1, "job number, async mode if larger than 1, maximum is 20")
)

func init() {
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        false,
		TimestampFormat: "15:04:05", // hour, time, sec only
		FieldsOrder:     []string{"M_ID", "IMG_ID"},
	})
	logrus.SetOutput(os.Stdout)
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "mirror":
		mirrorCmd.Parse(os.Args[2:])
		if *mirrorDebug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		logrus.Debugf("saveFile: %s", *saveFile)
		logrus.Debugf("saveArch: %s", *saveArch)
		logrus.Debugf("sourceReg: %s", *mirrorSourceReg)
		logrus.Debugf("destReg: %s", *mirrorDestReg)
		logrus.Debugf("mirrorJobs: %v", *mirrorJobs)
		logrus.Debugf("mirrorFailed: %v", *mirrorFailed)
		MirrorImages()
	case "save":
		saveCmd.Parse(os.Args[2:])
		if *saveDebug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		logrus.Debugf("mirrorFile: %s", *mirrorFile)
		logrus.Debugf("mirrorArch: %s", *mirrorArch)
		logrus.Debugf("saveSourceReg: %s", *saveSourceReg)
		logrus.Debugf("saveDestDir: %s", *saveDestDir)
		logrus.Debugf("saveFailed: %v", *saveFailed)
		logrus.Debugf("saveJobs: %v", *saveJobs)
		SaveImages()
	case "load":
		// loadCmd.Usage = func() {
		// 	fmt.Fprintf(loadCmd.Output(), "Usage: \n\t%s load [OPTIONS] file-name.tar.gz\n", os.Args[0])
		// 	fmt.Fprintf(loadCmd.Output(), "Options:\n")
		// 	loadCmd.PrintDefaults()
		// }
		loadCmd.Parse(os.Args[2:])
		if *loadDebug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		logrus.Debugf("loadFile: %v", *loadFile)
		logrus.Debugf("loadDestReg: %v", *loadDestReg)
		logrus.Debugf("loadFailed: %v", *loadFailed)
		logrus.Debugf("loadDebug: %v", *loadDebug)
		logrus.Debugf("loadJobs: %v", *loadJobs)
		LoadImages()
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
