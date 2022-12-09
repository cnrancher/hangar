package main

import (
	"fmt"
	"os"

	u "cnrancher.io/image-tools/utils"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

var (
	dockerUsername = os.Getenv("DOCKER_USERNAME")
	dockerPassword = os.Getenv("DOCKER_PASSWORD")
	dockerRegistry = os.Getenv("DOCKER_REGISTRY")
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
	case "version":
		showVersion()
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
	fmt.Printf("  load \t\tLoad image from saved tar.gz file.\n")
	fmt.Printf("  save \t\tSave image from source registry to tar.gz file.\n")
	fmt.Printf("  version \tShow version.\n")
}

func showVersion() {
	fmt.Printf("%s v%s\n", os.Args[0], u.VERSION)
}
