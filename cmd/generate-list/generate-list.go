package generatelist

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"

	"github.com/cnrancher/image-tools/pkg/rancher/chart"
	"github.com/cnrancher/image-tools/pkg/rancher/listgenerator"
	"github.com/cnrancher/image-tools/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	cmd               = flag.NewFlagSet("generate-list", flag.ExitOnError)
	cmdChart          = cmd.String("chart", "", "chart path/url")
	cmdSystemChart    = cmd.String("system-chart", "", "system chart path/url")
	cmdKDM            = cmd.String("kdm", "", "kdm path/url")
	cmdOutput         = cmd.String("o", "generated-list.txt", "generated image list path (linux and windows images)")
	cmdOutputLinux    = cmd.String("output-linux", "", "generated linux image list")
	cmdOutputWindows  = cmd.String("output-windows", "", "generated windows image list")
	cmdOutputSource   = cmd.String("output-source", "", "generate image list with image source")
	cmdRancherVersion = cmd.String("rancher", "v2.7.0", "rancher version (senmantic version with 'v' prefix)")
)

func Parse(args []string) {
	cmd.Parse(args)
}

func GenerateList() {
	if *cmdRancherVersion == "" {
		logrus.Error("rancher version not specified!")
		logrus.Error("Use '-rancher' option to specify the rancher version")
		os.Exit(1)
	}
	if *cmdOutput == "" {
		logrus.Error("output file not specified!")
		logrus.Error("Use '-o' option to specify the output file")
		os.Exit(1)
	}
	if *cmdChart == "" && *cmdSystemChart == "" && *cmdKDM == "" {
		logrus.Error("No input specified")
		cmd.Usage()
		os.Exit(1)
	}
	generator := listgenerator.Generator{}
	if *cmdKDM != "" {
		if _, err := url.ParseRequestURI(*cmdKDM); err != nil {
			generator.KDMURL = *cmdKDM
		} else {
			generator.KDMPath = *cmdKDM
		}
	}
	if *cmdChart != "" {
		if _, err := url.ParseRequestURI(*cmdChart); err != nil {
			generator.ChartURLs[*cmdChart] = chart.RepoTypeDefault
		} else {
			generator.ChartsPaths[*cmdChart] = chart.RepoTypeDefault
		}
	}
	if *cmdSystemChart != "" {
		if _, err := url.ParseRequestURI(*cmdSystemChart); err != nil {
			generator.ChartURLs[*cmdSystemChart] = chart.RepoTypeSystem
		} else {
			generator.ChartsPaths[*cmdSystemChart] = chart.RepoTypeSystem
		}
	}
	if err := generator.Generate(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	// merge windows images and linux images into one file
	imagesLinuxSet := map[string]bool{}
	imagesWindowsSet := map[string]bool{}
	var imagesSourceList = make([]string, 0,
		len(generator.GeneratedLinuxImages)+
			len(generator.GeneratedWindowsImages))

	for image := range generator.GeneratedLinuxImages {
		for source := range generator.GeneratedLinuxImages[image] {
			imagesLinuxSet[image] = true
			imagesSourceList = append(
				imagesSourceList, fmt.Sprintf("%s %s", image, source))
		}
	}
	for image := range generator.GeneratedWindowsImages {
		for source := range generator.GeneratedWindowsImages[image] {
			imagesWindowsSet[image] = true
			imagesSourceList = append(
				imagesSourceList, fmt.Sprintf("%s %s", image, source))
		}
	}
	var imagesList = make([]string, 0,
		len(imagesLinuxSet)+len(imagesWindowsSet))
	var imagesLinuxList = make([]string, 0, len(imagesLinuxSet))
	var imagesWindowsList = make([]string, 0, len(imagesWindowsSet))
	for img := range imagesLinuxSet {
		imagesLinuxList = append(imagesLinuxList, img)
		imagesList = append(imagesList, img)
	}
	for img := range imagesWindowsSet {
		imagesWindowsList = append(imagesWindowsList, img)
		imagesList = append(imagesList, img)
	}
	sort.Strings(imagesList)
	sort.Strings(imagesLinuxList)
	sort.Strings(imagesWindowsList)
	sort.Strings(imagesSourceList)
	if *cmdOutput != "" {
		err := utils.SaveSlice(*cmdOutput, imagesList)
		if err != nil {
			logrus.Error(err)
		}
	}
	if *cmdOutputLinux != "" {
		err := utils.SaveSlice(*cmdOutputLinux, imagesLinuxList)
		if err != nil {
			logrus.Error(err)
		}
	}
	if *cmdOutputWindows != "" {
		err := utils.SaveSlice(*cmdOutputWindows, imagesWindowsList)
		if err != nil {
			logrus.Error(err)
		}
	}
	if *cmdOutputSource != "" {
		err := utils.SaveSlice(*cmdOutputSource, imagesSourceList)
		if err != nil {
			logrus.Error(err)
		}
	}
}
