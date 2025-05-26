package commands

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	outputWindows  string
	outputSource   string
	outputVersions string
	rancherVersion string
	minKubeVersion string
	dev            bool
	tlsVerify      bool
	charts         []string
	systemCharts   []string
	autoYes        bool

	rke1Images          string
	rke2Images          string
	rke2WindowsImages   string
	k3sImages           string
	kdmRemoveDeprecated bool
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
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
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
	flags.StringVarP(&cc.output, "output", "o", "", "output linux image list file (default \"[RANCHER_VERSION]-images.txt\")")
	flags.StringVarP(&cc.outputWindows, "output-windows", "", "", "output the windows image list if specified")
	flags.StringVarP(&cc.outputSource, "output-source", "", "", "output the image list with image source if specified")
	flags.StringVarP(&cc.outputVersions, "output-versions", "", "", "output Rancher supported k8s versions (default \"[RANCHER_VERSION]-k8s-versions.txt\")")
	flags.StringVarP(&cc.rancherVersion, "rancher", "", "", "rancher version (semver with 'v' prefix) "+
		"(use '-ent' suffix to distinguish with Rancher Prime Manager GC) (required)")
	flags.StringVarP(&cc.minKubeVersion, "min-kube-version", "", "", "min kube version for RKE2/K3s when generate images, example: 'v1.28' (optional)")
	flags.BoolVarP(&cc.dev, "dev", "", false, "switch to dev branch/URL of charts & KDM data")
	flags.StringVarP(&cc.kdm, "kdm", "", "", "KDM file path or URL")
	flags.StringSliceVarP(&cc.charts, "chart", "", nil, "cloned chart repo path (URL not supported)")
	flags.StringSliceVarP(&cc.systemCharts, "system-chart", "", nil, "cloned system chart repo path (URL not supported)")
	flags.BoolVarP(&cc.kdmRemoveDeprecated, "kdm-remove-deprecated", "", true, "remove deprecated k3s/rke2 k8s versions from KDM")
	flags.StringVarP(&cc.rke1Images, "rke-images", "", "", "output KDM RKE linux image list if specified")
	flags.StringVarP(&cc.rke2Images, "rke2-images", "", "", "output KDM RKE2 linux image list if specified")
	flags.StringVarP(&cc.rke2WindowsImages, "rke2-windows-images", "", "", "output KDM RKE2 Windows image list if specified")
	flags.StringVarP(&cc.k3sImages, "k3s-images", "", "", "output KDM K3s linux image list if specified")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")

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
	if cc.outputVersions == "" {
		cc.outputVersions = cc.rancherVersion + "-versions.txt"
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
	option := &listgenerator.GeneratorOption{
		RancherVersion: cc.rancherVersion,
		MinKubeVersion: "",
		ChartsPaths:    make(map[string]chartimages.ChartRepoType),
		ChartURLs: make(map[string]struct {
			Type   chartimages.ChartRepoType
			Branch string
		}),
		InsecureSkipTLS:     !cc.tlsVerify,
		RemoveDeprecatedKDM: cc.kdmRemoveDeprecated,
	}

	if cc.minKubeVersion != "" {
		minKubeVersion := semver.MajorMinor(cc.minKubeVersion)
		option.MinKubeVersion = minKubeVersion
		if minKubeVersion == "" {
			return fmt.Errorf("invalid min-kube-version provided: %v",
				cc.minKubeVersion)
		}
	} else {
		switch {
		case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.7"):
			option.MinKubeVersion = "v1.23.0"
		case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.8"):
			option.MinKubeVersion = "v1.25.0"
		case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.9"):
			option.MinKubeVersion = "v1.27.0"
		case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.10"):
			option.MinKubeVersion = "v1.28.0"
		case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.11"):
			option.MinKubeVersion = "v1.30.0"
		case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.12"):
			option.MinKubeVersion = "v1.31.0"
		case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.13"):
			option.MinKubeVersion = "v1.32.0"
		default:
			option.MinKubeVersion = "v1.30.0"
		}
	}
	logrus.Infof("Min Kube-Version for Rancher [%v]: %v", cc.rancherVersion, option.MinKubeVersion)
	if cc.kdm != "" {
		if _, err := url.ParseRequestURI(cc.kdm); err != nil {
			option.KDMPath = cc.kdm
		} else {
			option.KDMURL = cc.kdm
		}
	}

	charts := cc.charts
	if len(charts) != 0 {
		for _, chart := range charts {
			if _, err := url.ParseRequestURI(chart); err != nil {
				logrus.Debugf("Add chart path to load images: %q", chart)
				option.ChartsPaths[chart] = chartimages.RepoTypeDefault
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
				option.ChartsPaths[chart] = chartimages.RepoTypeSystem
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
			addRancherPrimeCharts(cc.rancherVersion, option, dev)
			addRancherPrimeGCCharts(cc.rancherVersion, option, dev)
			addRancherPrimeGCSystemCharts(cc.rancherVersion, option, dev)
			addRancherPrimeManagerGCKontainerDriverMetadata(cc.rancherVersion, option, dev)
		} else {
			logrus.Debugf("Add Rancher Prime Manager charts & KDM to generate list")
			addRancherPrimeCharts(cc.rancherVersion, option, dev)
			addRancherPrimeSystemCharts(cc.rancherVersion, option, dev)
			addRancherPrimeKontainerDriverMetadata(cc.rancherVersion, option, dev)
		}
	}
	g, err := listgenerator.NewGenerator(option)
	if err != nil {
		return err
	}
	cc.generator = g

	return nil
}

func (cc *generateListCmd) run(ctx context.Context) error {
	err := cc.generator.Run(ctx)

	// Cleanup cache (if exists) after generate image list.
	cacheDir := filepath.Join(utils.HangarCacheDir(), utils.CacheCloneRepoDirectory)
	if err1 := os.RemoveAll(cacheDir); err1 != nil {
		logrus.Warnf("Failed to delete %q: %v", cacheDir, err1)
	}
	return err
}

func (cc *generateListCmd) finish() error {
	var (
		imagesLinuxList   = make([]string, 0)
		imagesWindowsList = make([]string, 0)
		imageSourcesList  = make([]string, 0)

		rke1LinuxImageList   = make([]string, 0)
		rke2LinuxImageList   = make([]string, 0)
		rke2WindowsImageList = make([]string, 0)
		k3sLinuxImageList    = make([]string, 0)

		rkeVersions  = make([]string, 0)
		rke2Versions = make([]string, 0)
		k3sVersions  = make([]string, 0)
	)

	var needUpdateWebhook bool

	if cc.isRPMGC {
		res, err := utils.SemverCompare(cc.rancherVersion, "v2.7.2")
		if err != nil {
			return fmt.Errorf("failed to compare version [%v] with [v2.7.2]: %w",
				cc.rancherVersion, err)
		}
		needUpdateWebhook = res > 0
	}
	for img := range cc.generator.LinuxImages {
		if needUpdateWebhook &&
			utils.GetImageName(img) == "rancher-webhook" &&
			utils.GetProjectName(img) == "rancher" {
			oldImg := img
			img = utils.ReplaceProjectName(img, "cnrancher")
			logrus.Infof("Replaced %q to %q", oldImg, img)
		}
		imgWithRegistry := img
		if cc.registry != "" {
			imgWithRegistry = utils.ConstructRegistry(img, cc.registry)
		}
		imagesLinuxList = append(imagesLinuxList, imgWithRegistry)
		imageSourcesList = append(imageSourcesList,
			fmt.Sprintf("%s %s", imgWithRegistry,
				getSourcesList(cc.generator.LinuxImages[img])))
	}
	for img := range cc.generator.WindowsImages {
		imgWithRegistry := img
		if cc.registry != "" {
			imgWithRegistry = utils.ConstructRegistry(img, cc.registry)
		}
		imagesWindowsList = append(imagesWindowsList, imgWithRegistry)
		imageSourcesList = append(imageSourcesList,
			fmt.Sprintf("%s %s", imgWithRegistry,
				getSourcesList(cc.generator.WindowsImages[img])))
	}
	for img := range cc.generator.RKE1LinuxImages {
		imgWithRegistry := img
		if cc.registry != "" {
			imgWithRegistry = utils.ConstructRegistry(img, cc.registry)
		}
		rke1LinuxImageList = append(rke1LinuxImageList, imgWithRegistry)
	}
	for img := range cc.generator.RKE2LinuxImages {
		imgWithRegistry := img
		if cc.registry != "" {
			imgWithRegistry = utils.ConstructRegistry(img, cc.registry)
		}
		rke2LinuxImageList = append(rke2LinuxImageList, imgWithRegistry)
	}
	for img := range cc.generator.RKE2WindowsImages {
		imgWithRegistry := img
		if cc.registry != "" {
			imgWithRegistry = utils.ConstructRegistry(img, cc.registry)
		}
		rke2WindowsImageList = append(rke2WindowsImageList, imgWithRegistry)
	}
	for img := range cc.generator.K3sLinuxImages {
		imgWithRegistry := img
		if cc.registry != "" {
			imgWithRegistry = utils.ConstructRegistry(img, cc.registry)
		}
		k3sLinuxImageList = append(k3sLinuxImageList, imgWithRegistry)
	}
	for v := range cc.generator.RKE1Versions {
		rkeVersions = append(rkeVersions, v)
	}
	for v := range cc.generator.RKE2Versions {
		rke2Versions = append(rke2Versions, v)
	}
	for v := range cc.generator.K3sVersions {
		k3sVersions = append(k3sVersions, v)
	}
	sort.Strings(imagesLinuxList)
	sort.Strings(imagesWindowsList)
	sort.Strings(imageSourcesList)
	sort.Strings(rke1LinuxImageList)
	sort.Strings(rke2LinuxImageList)
	sort.Strings(rke2WindowsImageList)
	sort.Strings(k3sLinuxImageList)

	sort.Slice(rkeVersions, func(i, j int) bool {
		ok, _ := utils.SemverCompare(rkeVersions[i], rkeVersions[j])
		return ok < 0
	})
	sort.Slice(rke2Versions, func(i, j int) bool {
		ok, _ := utils.SemverCompare(rke2Versions[i], rke2Versions[j])
		return ok < 0
	})
	sort.Slice(k3sVersions, func(i, j int) bool {
		ok, _ := utils.SemverCompare(k3sVersions[i], k3sVersions[j])
		return ok < 0
	})

	if cc.output != "" {
		err := cc.saveSlice(signalContext, cc.output, imagesLinuxList)
		if err != nil {
			return fmt.Errorf("failed to write file %q: %w", cc.output, err)
		}
		logrus.Infof("Exported Rancher linux images into %v", cc.output)
	}
	if cc.outputWindows != "" {
		err := cc.saveSlice(signalContext, cc.outputWindows, imagesWindowsList)
		if err != nil {
			return fmt.Errorf("failed to write file %q: %w", cc.outputWindows, err)
		}
		logrus.Infof("Exported Rancher windows images into %v", cc.outputWindows)
	}
	if cc.outputSource != "" {
		err := cc.saveSlice(signalContext, cc.outputSource, imageSourcesList)
		if err != nil {
			return fmt.Errorf("failed to write file %q: %w", cc.outputSource, err)
		}
		logrus.Infof("Exported Rancher images sources into %v", cc.outputSource)
	}
	if cc.outputVersions != "" {
		var versions []string
		versions = append(versions, fmt.Sprintf("K3s, RKE2, RKE versions for Rancher %v:", cc.rancherVersion))
		versions = append(versions, "")
		versions = append(versions, "K3s Versions:")
		versions = append(versions, k3sVersions...)
		versions = append(versions, "")
		versions = append(versions, "RKE2 Versions:")
		versions = append(versions, rke2Versions...)
		versions = append(versions, "")
		versions = append(versions, "RKE Versions:")
		versions = append(versions, rkeVersions...)
		err := cc.saveSlice(signalContext, cc.outputVersions, versions)
		if err != nil {
			return fmt.Errorf("failed to write file %q: %w", cc.outputVersions, err)
		}
		logrus.Infof("Exported Rancher supported versions into %v", cc.outputVersions)
	}
	if cc.rke1Images != "" {
		err := cc.saveSlice(signalContext, cc.rke1Images, rke1LinuxImageList)
		if err != nil {
			return fmt.Errorf("failed to write file %q: %w", cc.rke1Images, err)
		}
		logrus.Infof("Exported RKE1 Linux images into %v", cc.rke1Images)
	}
	if cc.k3sImages != "" {
		err := cc.saveSlice(signalContext, cc.k3sImages, k3sLinuxImageList)
		if err != nil {
			return fmt.Errorf("failed to write file %q: %w", cc.k3sImages, err)
		}
		logrus.Infof("Exported K3s Linux images into %v", cc.k3sImages)
	}
	if cc.rke2Images != "" {
		err := cc.saveSlice(signalContext, cc.rke2Images, rke2LinuxImageList)
		if err != nil {
			return fmt.Errorf("failed to write file %q: %w", cc.rke2Images, err)
		}
		logrus.Infof("Exported RKE2 Linux images into %v", cc.rke2Images)
	}
	if cc.rke2WindowsImages != "" {
		err := cc.saveSlice(signalContext, cc.rke2WindowsImages, rke2WindowsImageList)
		if err != nil {
			return fmt.Errorf("failed to write file %q: %w", cc.rke2WindowsImages, err)
		}
		logrus.Infof("Exported RKE2 Linux images into %v", cc.rke2WindowsImages)
	}
	return nil
}

func getSourcesList(imageSources map[string]bool) string {
	var sources = []string{}
	for source := range imageSources {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	return strings.Join(sources, ",")
}

func (cc *generateListCmd) saveSlice(ctx context.Context, name string, data []string) error {
	if err := utils.CheckFileExistsPrompt(ctx, name, cc.autoYes); err != nil {
		return err
	}

	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(strings.Join(data, "\n"))
	if err != nil {
		return err
	}
	return nil
}
