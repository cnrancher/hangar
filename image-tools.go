package main

import (
	"fmt"
	"os"

	loadCMD "cnrancher.io/image-tools/cmd/load"
	mirrorCMD "cnrancher.io/image-tools/cmd/mirror"
	saveCMD "cnrancher.io/image-tools/cmd/save"
	u "cnrancher.io/image-tools/utils"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
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
		mirrorCMD.CMD.Parse(os.Args[2:])
		mirrorCMD.MirrorImages()
	case "save":
		saveCMD.CMD.Parse(os.Args[2:])
		saveCMD.SaveImages()
	case "load":
		loadCMD.CMD.Parse(os.Args[2:])
		loadCMD.LoadImages()
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
