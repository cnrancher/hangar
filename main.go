package main

import (
	"fmt"
	"os"

	convertCMD "cnrancher.io/image-tools/cmd/convert"
	loadCMD "cnrancher.io/image-tools/cmd/load"
	mirrorCMD "cnrancher.io/image-tools/cmd/mirror"
	mirrorValidateCMD "cnrancher.io/image-tools/cmd/mirror-validate"
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
		mirrorCMD.Parse(os.Args[2:])
		mirrorCMD.MirrorImages()
	case "save":
		saveCMD.Parse(os.Args[2:])
		saveCMD.SaveImages()
	case "load":
		loadCMD.Parse(os.Args[2:])
		loadCMD.LoadImages()
	case "convert-list":
		convertCMD.Parse(os.Args[2:])
		convertCMD.Convert()
	case "mirror-validate":
		mirrorValidateCMD.Parse(os.Args[2:])
		mirrorValidateCMD.ValidateImages()
	case "-v":
		fallthrough
	case "--version":
		fallthrough
	case "version":
		showVersion()
	case "-h", "--help":
		showHelp()
		os.Exit(0)
	default:
		logrus.Errorf("unrecognized command %q", os.Args[1])
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf("Usage:\t%s COMMAND [OPTIONS]\n", os.Args[0])
	fmt.Println()
	fmt.Printf("Run '%s COMMAND --help' for more information on a command.\n", os.Args[0])
	fmt.Println()
	fmt.Printf("Commands: \n")
	fmt.Printf("  mirror \t\tMirror image from source registry to destination registry.\n")
	fmt.Printf("  load \t\t\tLoad image from saved tar.gz file.\n")
	fmt.Printf("  save \t\t\tSave image from source registry to tar.gz file.\n")
	fmt.Printf("  convert-list \t\tConvert image list to 'mirror' format.\n")
	fmt.Printf("  mirror-validate \tValidate mirrored images.\n")
	fmt.Printf("  version \t\tShow version.\n")
}

func showVersion() {
	fmt.Printf("%s v%s\n", os.Args[0], u.VERSION)
}
