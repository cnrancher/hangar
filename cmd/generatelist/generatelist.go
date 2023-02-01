package generatelist

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/cnrancher/image-tools/cmd"
	"github.com/cnrancher/image-tools/pkg/rancher/chartimages"
	"github.com/cnrancher/image-tools/pkg/rancher/listgenerator"
	u "github.com/cnrancher/image-tools/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	cmdRegistry       string
	cmdKDM            string
	cmdOutput         string
	cmdOutputLinux    string
	cmdOutputWindows  string
	cmdOutputSource   string
	cmdRancherVersion string
	cmdKubeVersion    string
	cmdDebug          bool
	cmdCharts         cmd.StringSlice
	cmdSystemCharts   cmd.StringSlice
	flagSet           = flag.NewFlagSet("generate-list", flag.ExitOnError)

	IsRPMGC bool
)

func Parse(args []string) {
	flagSet.StringVar(&cmdRegistry, "registry", "", "override the registry url")
	flagSet.StringVar(&cmdKDM, "kdm", "", "kdm path/url")
	flagSet.StringVar(&cmdOutput, "o", "generated-list.txt", "generated image list path (linux and windows images)")
	flagSet.StringVar(&cmdOutputLinux, "output-linux", "", "generated linux image list")
	flagSet.StringVar(&cmdOutputWindows, "output-windows", "", "generated windows image list")
	flagSet.StringVar(&cmdOutputSource, "output-source", "", "generate image list with image source")
	flagSet.StringVar(&cmdRancherVersion, "rancher", "v2.7.0-ent", "rancher version (semantic version with 'v' prefix)")
	flagSet.StringVar(&cmdKubeVersion, "kubeversion", "v1.21.0", "minimum kuber version (semantic version with 'v' prefix)")
	flagSet.BoolVar(&cmdDebug, "debug", false, "enable the debug output")

	flagSet.Var(&cmdCharts, "chart", "chart path")
	flagSet.Var(&cmdSystemCharts, "system-chart", "system chart path")
	flagSet.Parse(args)
}

func GenerateList() {
	if cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if cmdOutput == "" {
		logrus.Error("output file not specified!")
		logrus.Error("Use '-o' option to specify the output file")
		os.Exit(1)
	}
	// if len(cmdCharts) == 0 && len(cmdSystemCharts) == 0 && cmdKDM == "" {
	// 	logrus.Error("No input specified")
	// 	logrus.Error("Please use '-kdm' or '-chart' or '-system-chart' " +
	// 		"to specify the input resource")
	// 	flagSet.Usage()
	// 	os.Exit(1)
	// }
	if cmdKubeVersion == "" {
		logrus.Error("minimum kube version not specified!")
		logrus.Error("Use '-kubeversion' to specify the min kube version")
		os.Exit(1)
	}
	if !strings.HasPrefix(cmdKubeVersion, "v") {
		cmdKubeVersion = "v" + cmdKubeVersion
	}
	if cmdRancherVersion == "" {
		logrus.Error("rancher version not specified!")
		logrus.Error("Use '-rancher' option to specify the rancher version")
		logrus.Error("Version format example: 'v2.7.0'")
		os.Exit(1)
	}
	if !strings.HasPrefix(cmdRancherVersion, "v") {
		cmdRancherVersion = "v" + cmdRancherVersion
	}
	if strings.HasSuffix(cmdRancherVersion, "-ent") {
		IsRPMGC = true
		cmdRancherVersion = strings.TrimSuffix(cmdRancherVersion, "-ent")
	}
	generator := listgenerator.Generator{
		RancherVersion: cmdRancherVersion,
		MinKubeVersion: cmdKubeVersion,
		ChartsPaths:    make(map[string]chartimages.ChartRepoType),
		ChartURLs: make(map[string]struct {
			Type   chartimages.ChartRepoType
			Branch string
		}),
	}
	if cmdKDM != "" {
		if _, err := url.ParseRequestURI(cmdKDM); err != nil {
			generator.KDMPath = cmdKDM
		} else {
			generator.KDMURL = cmdKDM
		}
	}
	if len(cmdCharts) != 0 {
		for _, chart := range cmdCharts {
			if _, err := url.ParseRequestURI(chart); err != nil {
				generator.ChartsPaths[chart] = chartimages.RepoTypeDefault
			} else {
				generator.ChartURLs[chart] = struct {
					Type   chartimages.ChartRepoType
					Branch string
				}{
					Type:   chartimages.RepoTypeDefault,
					Branch: "", // use default branch
				}
			}
		}
	}
	if len(cmdSystemCharts) != 0 {
		for _, chart := range cmdSystemCharts {
			if _, err := url.ParseRequestURI(chart); err != nil {
				generator.ChartsPaths[chart] = chartimages.RepoTypeSystem
			} else {
				generator.ChartURLs[chart] = struct {
					Type   chartimages.ChartRepoType
					Branch string
				}{
					Type:   chartimages.RepoTypeSystem,
					Branch: "",
				}
			}
		}
	}
	// if no input specified, use default values
	if cmdKDM == "" && len(cmdCharts) == 0 && len(cmdSystemCharts) == 0 {
		AddRPMCharts(cmdRancherVersion, &generator)
		if IsRPMGC {
			AddRPMGCCharts(cmdRancherVersion, &generator)
			AddRPM_GC_KDM(cmdRancherVersion, &generator)
		} else {
			AddRPM_KDM(cmdRancherVersion, &generator)
		}
	}
	if err := generator.Generate(); err != nil {
		logrus.Fatal(err)
	}
	// merge windows images and linux images into one file
	imagesLinuxSet := map[string]bool{}
	imagesWindowsSet := map[string]bool{}
	var imageSources = make([]string, 0,
		len(generator.GeneratedLinuxImages)+
			len(generator.GeneratedWindowsImages))

	for image := range generator.GeneratedLinuxImages {
		imgWithRegistry := image
		if cmdRegistry != "" {
			imgWithRegistry = u.ConstructRegistry(image, cmdRegistry)
		}
		imagesLinuxSet[imgWithRegistry] = true
		imageSources = append(imageSources,
			fmt.Sprintf("%s %s", imgWithRegistry,
				getSourcesList(generator.GeneratedLinuxImages[image])))
	}
	for image := range generator.GeneratedWindowsImages {
		imgWithRegistry := image
		if cmdRegistry != "" {
			imgWithRegistry = u.ConstructRegistry(image, cmdRegistry)
		}
		imagesWindowsSet[imgWithRegistry] = true
		imageSources = append(imageSources,
			fmt.Sprintf("%s %s", imgWithRegistry,
				getSourcesList(generator.GeneratedWindowsImages[image])))
	}
	var imagesAllSet = map[string]bool{}
	var imagesLinuxList = make([]string, 0, len(imagesLinuxSet))
	var imagesWindowsList = make([]string, 0, len(imagesWindowsSet))
	for img := range imagesLinuxSet {
		imagesLinuxList = append(imagesLinuxList, img)
		imagesAllSet[img] = true
	}
	for img := range imagesWindowsSet {
		imagesWindowsList = append(imagesWindowsList, img)
		imagesAllSet[img] = true
	}
	var imagesList = make([]string, 0,
		len(imagesLinuxSet)+len(imagesWindowsSet))
	for img := range imagesAllSet {
		imagesList = append(imagesList, img)
	}
	sort.Strings(imagesList)
	sort.Strings(imagesLinuxList)
	sort.Strings(imagesWindowsList)
	sort.Strings(imageSources)
	if cmdOutput != "" {
		err := u.SaveSlice(cmdOutput, imagesList)
		if err != nil {
			logrus.Error(err)
		}
	}
	if cmdOutputLinux != "" {
		err := u.SaveSlice(cmdOutputLinux, imagesLinuxList)
		if err != nil {
			logrus.Error(err)
		}
	}
	if cmdOutputWindows != "" {
		err := u.SaveSlice(cmdOutputWindows, imagesWindowsList)
		if err != nil {
			logrus.Error(err)
		}
	}
	if cmdOutputSource != "" {
		err := u.SaveSlice(cmdOutputSource, imageSources)
		if err != nil {
			logrus.Error(err)
		}
	}
}

func getSourcesList(imageSources map[string]bool) string {
	var sources []string
	for source := range imageSources {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	return strings.Join(sources, ",")
}
