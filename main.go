package main

import (
	"fmt"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"

	convertCMD "github.com/cnrancher/hangar/cmd/convert"
	generateListCMD "github.com/cnrancher/hangar/cmd/generatelist"
	loadCMD "github.com/cnrancher/hangar/cmd/load"
	loadValidateCMD "github.com/cnrancher/hangar/cmd/loadvalidate"
	mirrorCMD "github.com/cnrancher/hangar/cmd/mirror"
	mirrorValidateCMD "github.com/cnrancher/hangar/cmd/mirrorvalidate"
	saveCMD "github.com/cnrancher/hangar/cmd/save"
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
		mirrorValidateCMD.MirrorValidate()
	case "load-validate":
		loadValidateCMD.Parse(os.Args[2:])
		loadValidateCMD.LoadValidate()
	case "generate-list":
		generateListCMD.Parse(os.Args[2:])
		generateListCMD.GenerateList()
	case "-v", "--version", "version":
		showVersion()
	case "-h", "--help", "help":
		showHelp()
	default:
		logrus.Errorf("unrecognized command %q", os.Args[1])
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf("Usage:\t%s COMMAND [OPTIONS]\n", os.Args[0])
	fmt.Printf("\n")
	fmt.Printf("Run '%s COMMAND --help' for more information on a command.\n", os.Args[0])
	fmt.Printf("\n")
	fmt.Printf("Commands: \n")
	fmt.Printf("  mirror \t\tMirror image from source registry to destination registry.\n")
	fmt.Printf("  save \t\t\tSave image from source registry to local file.\n")
	fmt.Printf("  load \t\t\tLoad image from saved local file.\n")
	fmt.Printf("  convert-list \t\tConvert image list to 'mirror' format.\n")
	fmt.Printf("  mirror-validate \tValidate mirrored images.\n")
	fmt.Printf("  load-validate \tValidate loaded images.\n")
	fmt.Printf("  generate-list \tGenerate list from KDM data/charts repo.\n")
	fmt.Printf("  version \t\tShow version.\n")
	fmt.Printf("\n")
	fmt.Printf("More detailed usage docs is available in <%s/tree/main/docs/>\n", utils.HangarGitHubURL)
}

func showVersion() {
	if utils.GitCommit != "" {
		fmt.Printf("%s %s - %s\n", os.Args[0], utils.Version, utils.GitCommit)
	} else {
		fmt.Printf("%s %s\n", os.Args[0], utils.Version)
	}
}
