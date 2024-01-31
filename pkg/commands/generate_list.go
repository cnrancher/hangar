package commands

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/cnrancher/hangar/pkg/rancher/chartimages"
	"github.com/cnrancher/hangar/pkg/rancher/listgenerator"
	"github.com/cnrancher/hangar/pkg/utils"
	commonFlag "github.com/containers/common/pkg/flag"
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
	dev            bool
	tlsVerify      commonFlag.OptionalBool
	charts         []string
	systemCharts   []string
	autoYes        bool
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
	flags.BoolVarP(&cc.dev, "dev", "", false, "switch to dev branch/URL of charts & KDM data")
	flags.StringVarP(&cc.kdm, "kdm", "", "", "KDM file path or URL")
	flags.StringSliceVarP(&cc.charts, "chart", "", nil, "cloned chart repo path (URL not supported)")
	flags.StringSliceVarP(&cc.systemCharts, "system-chart", "", nil, "cloned system chart repo path (URL not supported)")
	commonFlag.OptionalBoolFlag(flags, &cc.tlsVerify, "tls-verify", "require HTTPS and verify certificates")
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
	cc.generator = &listgenerator.Generator{
		RancherVersion: cc.rancherVersion,
		MinKubeVersion: "",
		ChartsPaths:    make(map[string]chartimages.ChartRepoType),
		ChartURLs: make(map[string]struct {
			Type   chartimages.ChartRepoType
			Branch string
		}),
	}
	if cc.tlsVerify.Present() {
		cc.generator.InsecureSkipVerify = !cc.tlsVerify.Value()
	}
	switch {
	case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.6"):
		cc.generator.MinKubeVersion = "v1.21.0"
	case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.7"):
		cc.generator.MinKubeVersion = "v1.23.0"
	case utils.SemverMajorMinorEqual(cc.rancherVersion, "v2.8"):
		cc.generator.MinKubeVersion = "v1.25.0"
	default:
		cc.generator.MinKubeVersion = "v0.1.0"
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
	var (
		imagesLinuxList   = make([]string, 0)
		imagesWindowsList = make([]string, 0)
		imageSourcesList  = make([]string, 0)

		rkeVersions  = make([]string, 0)
		rke2Versions = make([]string, 0)
		k3sVersions  = make([]string, 0)
	)

	for img := range cc.generator.LinuxImages {
		res, err := utils.SemverCompare(cc.rancherVersion, "v2.7.2")
		if err != nil {
			return fmt.Errorf("failed to compare version [%v] with [v2.7.2]: %w",
				cc.rancherVersion, err)
		}
		if cc.isRPMGC && res >= 0 {
			if utils.GetImageName(img) == "rancher-webhook" &&
				utils.GetProjectName(img) == "rancher" {
				oldImg := img
				img = utils.ReplaceProjectName(img, "cnrancher")
				logrus.Infof("Replaced %q to %q", oldImg, img)
			}
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
