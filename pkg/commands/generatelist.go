package commands

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/rancher/chartimages"
	"github.com/cnrancher/hangar/pkg/rancher/listgenerator"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

type generateListOpts struct {
	registry       string
	kdm            string
	output         string
	outputLinux    string
	outputWindows  string
	outputSource   string
	rancherVersion string
	dev            bool
	charts         []string
	systemCharts   []string
}

type generateListCmd struct {
	*baseCmd
	*generateListOpts

	isRPMGC   bool
	generator *listgenerator.Generator
}

func newGenerateListCmd() *generateListCmd {
	cc := &generateListCmd{
		generateListOpts: new(generateListOpts),
	}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "generate-list",
		Short: "Generate Rancher image list file",
		Long: `'generate-list' generates an image list and k8s version list from KDM data and Chart repos of Rancher.

Generate the image list by simply specifying the Rancher version:

    hangar generate-list --rancher="v2.8.0"

You can also download the KDM JSON file and clone chart repos manually:

    hangar generate-list \
        --rancher="v2.8.0" \
        --chart="./chart-repo-dir" \
        --system-chart="./system-chart-repo-dir" \
        --kdm="./kdm-data.json"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cc.baseCmd.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("debug output enabled")
				logrus.Debugf("%v", utils.PrintObject(cmdconfig.Get("")))
			}

			if err := cc.setupFlags(); err != nil {
				return err
			}
			if err := cc.prepareGenerator(); err != nil {
				return err
			}
			if err := cc.run(signalContext); err != nil {
				return err
			}
			if err := cc.finish(); err != nil {
				return err
			}

			return nil
		},
	})
	flags := cc.baseCmd.cmd.PersistentFlags()
	flags.StringVarP(&cc.registry, "registry", "", "", "customize the registry URL of the generated image list")
	flags.StringVarP(&cc.kdm, "kdm", "", "", "KDM file path or URL (optional)")
	flags.StringVarP(&cc.output, "output", "o", "", "output generated image list file (default \"[RANCHER_VERSION]-images.txt\")")
	flags.StringVarP(&cc.outputLinux, "output-linux", "", "", "output the linux image list if specified")
	flags.StringVarP(&cc.outputWindows, "output-windows", "", "", "output the windows image list if specified")
	flags.StringVarP(&cc.outputSource, "output-source", "", "", "output the image list with image source if specified")
	flags.StringVarP(&cc.rancherVersion, "rancher", "", "", "rancher version (semver with 'v' prefix) "+
		"(use '-ent' suffix to distinguish with Rancher Prime Manager GC) (required)")
	flags.BoolVarP(&cc.dev, "dev", "", false, "switch to dev branch/URL of charts & KDM data")
	flags.StringSliceVarP(&cc.charts, "chart", "", nil, "cloned chart repo path (URL not supported)")
	flags.StringSliceVarP(&cc.systemCharts, "system-chart", "", nil, "cloned system chart repo path (URL not supported)")

	return cc
}

func (cc *generateListCmd) setupFlags() error {
	if cc.rancherVersion == "" {
		return fmt.Errorf("rancher version not specified, use '--rancher' to specify the rancher version")
	}
	if !strings.HasPrefix(cc.rancherVersion, "v") {
		cc.rancherVersion = "v" + cc.rancherVersion
	}
	if cc.output == "" {
		cc.output = cc.rancherVersion + "-images.txt"
	}
	if strings.Contains(cc.rancherVersion, "-ent") {
		logrus.Infof("Set to Rancher Prime Manager GC version")
		cc.isRPMGC = true
		v := strings.Split(cc.rancherVersion, "-ent")
		cc.rancherVersion = v[0]
	}
	if !semver.IsValid(cc.rancherVersion) {
		return fmt.Errorf("%q is not a valid semver version", cc.rancherVersion)
	}

	return nil
}

func (cc *generateListCmd) prepareGenerator() error {
	cc.generator = &listgenerator.Generator{
		RancherVersion: cc.rancherVersion,
		MinKubeVersion: "",
		ChartsPaths:    make(map[string]chartimages.ChartRepoType),
		ChartURLs: make(map[string]struct {
			Type   chartimages.ChartRepoType
			Branch string
		}),
	}
	switch {
	case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.5"):
		cc.generator.MinKubeVersion = ""
	case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.6"):
		cc.generator.MinKubeVersion = "v1.21.0"
	case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.7"):
		cc.generator.MinKubeVersion = "v1.23.0"
	case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.8"):
		cc.generator.MinKubeVersion = "v1.25.0"
	}
	if cc.kdm != "" {
		if _, err := url.ParseRequestURI(cc.kdm); err != nil {
			cc.generator.KDMPath = cc.kdm
		} else {
			cc.generator.KDMURL = cc.kdm
		}
	}

	charts := cc.charts
	if len(charts) != 0 {
		for _, chart := range charts {
			if _, err := url.ParseRequestURI(chart); err != nil {
				logrus.Debugf("Add chart path to load images: %q", chart)
				cc.generator.ChartsPaths[chart] = chartimages.RepoTypeDefault
			} else {
				// cc.generator.ChartURLs[chart] = struct {
				// 	Type   chartimages.ChartRepoType
				// 	Branch string
				// }{
				// 	Type:   chartimages.RepoTypeDefault,
				// 	Branch: "", // use default branch
				// }
				return fmt.Errorf("chart url is not supported, please provide the cloned chart path")
			}
		}
	}
	systemCharts := cc.systemCharts
	if len(systemCharts) != 0 {
		for _, chart := range systemCharts {
			if _, err := url.ParseRequestURI(chart); err != nil {
				logrus.Debugf("Add system chart path to load images: %q", chart)
				cc.generator.ChartsPaths[chart] = chartimages.RepoTypeSystem
			} else {
				return fmt.Errorf("chart url is not supported, please provide the cloned chart path")
			}
		}
	}
	dev := cc.dev
	if cc.kdm == "" && len(charts) == 0 && len(systemCharts) == 0 {
		if dev {
			logrus.Info("Using branch: dev")
		} else {
			logrus.Info("Using branch: release")
		}
		if cc.isRPMGC {
			logrus.Debugf("Add Rancher Prime Manager GC charts & KDM to generate list")
			addRPMCharts(cc.rancherVersion, cc.generator, dev)
			addRPMGCCharts(cc.rancherVersion, cc.generator, dev)
			addRPMGCSystemCharts(cc.rancherVersion, cc.generator, dev)
			addRancherPrimeManagerGCKontainerDriverMetadata(cc.rancherVersion, cc.generator, dev)
		} else {
			logrus.Debugf("Add Rancher Prime Manager charts & KDM to generate list")
			addRPMCharts(cc.rancherVersion, cc.generator, dev)
			addRPMSystemCharts(cc.rancherVersion, cc.generator, dev)
			addRancherPrimeManagerKontainerDriverMetadata(cc.rancherVersion, cc.generator, dev)
		}
	}

	return nil
}

func (cc *generateListCmd) run(ctx context.Context) error {
	err := cc.generator.Generate(ctx)

	// Cleanup cache (if exists) after generate image list.
	cacheDir := path.Join(utils.CacheDir(), utils.CacheCloneRepoDirectory)
	if err1 := os.RemoveAll(cacheDir); err1 != nil {
		logrus.Warnf("Failed to delete %q: %v", cacheDir, err1)
	}
	return err
}

func (cc *generateListCmd) finish() error {
	// merge windows images and linux images into one file
	imagesLinuxSet := map[string]bool{}
	imagesWindowsSet := map[string]bool{}
	var imageSources = make([]string, 0,
		len(cc.generator.GeneratedLinuxImages)+
			len(cc.generator.GeneratedWindowsImages))

	registry := cc.registry
	for image := range cc.generator.GeneratedLinuxImages {
		imgWithRegistry := image
		if registry != "" {
			imgWithRegistry = utils.ConstructRegistry(image, registry)
		}
		imagesLinuxSet[imgWithRegistry] = true
		imageSources = append(imageSources,
			fmt.Sprintf("%s %s", imgWithRegistry,
				getSourcesList(cc.generator.GeneratedLinuxImages[image])))
	}
	for image := range cc.generator.GeneratedWindowsImages {
		imgWithRegistry := image
		if registry != "" {
			imgWithRegistry = utils.ConstructRegistry(image, registry)
		}
		imagesWindowsSet[imgWithRegistry] = true
		imageSources = append(imageSources,
			fmt.Sprintf("%s %s", imgWithRegistry,
				getSourcesList(cc.generator.GeneratedWindowsImages[image])))
	}
	var imagesAllSet = map[string]bool{}
	var imagesLinuxList = make([]string, 0, len(imagesLinuxSet))
	var imagesWindowsList = make([]string, 0, len(imagesWindowsSet))
	for img := range imagesLinuxSet {
		res, err := utils.SemverCompare(cc.rancherVersion, "v2.7.2")
		if cc.isRPMGC && err == nil && res >= 0 {
			if utils.GetImageName(img) == "rancher-webhook" &&
				utils.GetProjectName(img) == "rancher" {
				oldImg := img
				img = utils.ReplaceProjectName(img, "cnrancher")
				logrus.Infof("Replaced %q to %q", oldImg, img)
			}
		} else if err != nil {
			logrus.Error(err)
		}
		imagesLinuxList = append(imagesLinuxList, img)
		imagesAllSet[img] = true
	}
	for img := range imagesWindowsSet {
		res, err := utils.SemverCompare(cc.rancherVersion, "v2.7.2")
		if cc.isRPMGC && err == nil && res >= 0 {
			if utils.GetImageName(img) == "rancher-webhook" &&
				utils.GetProjectName(img) == "rancher" {
				oldImg := img
				img = utils.ReplaceProjectName(img, "cnrancher")
				logrus.Infof("Replaced %q to %q", oldImg, img)
			}
		} else if err != nil {
			logrus.Error(err)
		}
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
	if cc.output != "" {
		err := utils.SaveSlice(cc.output, imagesList)
		if err != nil {
			logrus.Error(err)
		}
	}
	if cc.outputLinux != "" {
		err := utils.SaveSlice(cc.outputLinux, imagesLinuxList)
		if err != nil {
			logrus.Error(err)
		}
	}
	if cc.outputWindows != "" {
		err := utils.SaveSlice(cc.outputWindows, imagesWindowsList)
		if err != nil {
			logrus.Error(err)
		}
	}
	if cc.outputSource != "" {
		err := utils.SaveSlice(cc.outputSource, imageSources)
		if err != nil {
			logrus.Error(err)
		}
	}
	return nil
}

func getSourcesList(imageSources map[string]bool) string {
	var sources []string
	for source := range imageSources {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	return strings.Join(sources, ",")
}
