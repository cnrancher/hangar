package commands

import (
	"github.com/cnrancher/hangar/pkg/rancher/chartimages"
	"github.com/cnrancher/hangar/pkg/rancher/listgenerator"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

var (
	// map[version]map[url][branch]
	RancherPrimeManagerGCCharts = map[string]map[string]string{
		"v2.8": {
			// pandaria-catalog
			"https://github.com/cnrancher/pandaria-catalog": "release/v2.8",
		},
		"v2.7": {
			// pandaria-catalog
			"https://github.com/cnrancher/pandaria-catalog": "release/v2.7",
		},
		"v2.6": {
			// pandaria-catalog
			"https://github.com/cnrancher/pandaria-catalog": "release/v2.6",
		},
		"v2.5": {
			// pandaria-catalog
			"https://github.com/cnrancher/pandaria-catalog": "release/v2.5",
		},
	}

	// map[version]map[url][branch]
	RancherPrimeManagerGCSystemCharts = map[string]map[string]string{
		"v2.8": {
			// system-chart
			"https://github.com/cnrancher/system-charts": "release-v2.8-ent",
		},
		"v2.7": {
			// system-chart
			"https://github.com/cnrancher/system-charts": "release-v2.7-ent",
		},
		"v2.6": {
			// system-chart
			"https://github.com/cnrancher/system-charts": "release-v2.6-ent",
		},
		"v2.5": {
			// system-chart
			"https://github.com/cnrancher/system-charts": "release-v2.5-ent",
		},
	}

	// map[version]map[url][branch]
	RancherPrimeManagerGCChartsDEV = map[string]map[string]string{
		"v2.8": {
			// pandaria-catalog
			"https://github.com/cnrancher/pandaria-catalog": "dev/v2.8",
		},
		"v2.7": {
			// pandaria-catalog
			"https://github.com/cnrancher/pandaria-catalog": "dev/v2.7",
		},
		"v2.6": {
			// pandaria-catalog
			"https://github.com/cnrancher/pandaria-catalog": "dev/v2.6",
		},
		"v2.5": {
			// pandaria-catalog
			"https://github.com/cnrancher/pandaria-catalog": "dev/v2.5",
		},
	}

	// map[version]map[url][branch]
	RancherPrimeManagerGCSystemChartsDEV = map[string]map[string]string{
		"v2.8": {
			// system-chart
			"https://github.com/cnrancher/system-charts": "dev-v2.8",
		},
		"v2.7": {
			// system-chart
			"https://github.com/cnrancher/system-charts": "dev-v2.7",
		},
		"v2.6": {
			// system-chart
			"https://github.com/cnrancher/system-charts": "dev-v2.6",
		},
		"v2.5": {
			// system-chart
			"https://github.com/cnrancher/system-charts": "dev-v2.5",
		},
	}

	// map[version]map[url][branch]
	RancherPrimeManagerCharts = map[string]map[string]string{
		"v2.8": {
			// rancher-charts
			"https://github.com/rancher/charts": "release-v2.8",
		},
		"v2.7": {
			// rancher-charts
			"https://github.com/rancher/charts": "release-v2.7",
		},
		"v2.6": {
			// rancher-charts
			"https://github.com/rancher/charts": "release-v2.6",
		},
		"v2.5": {
			// system-chart
			"https://github.com/rancher/system-charts": "release-v2.5",
			// rancher-charts
			"https://github.com/rancher/charts": "release-v2.5",
		},
	}

	// map[version]map[url][branch]
	RancherPrimeManagerSystemCharts = map[string]map[string]string{
		"v2.8": {
			// system-chart
			"https://github.com/rancher/system-charts": "release-v2.8",
		},
		"v2.7": {
			// system-chart
			"https://github.com/rancher/system-charts": "release-v2.7",
		},
		"v2.6": {
			// system-chart
			"https://github.com/rancher/system-charts": "release-v2.6",
		},
		"v2.5": {
			// system-chart
			"https://github.com/rancher/system-charts": "release-v2.5",
		},
	}

	// map[version]map[url][branch]
	RancherPrimeManagerChartsDEV = map[string]map[string]string{
		"v2.8": {
			// rancher-charts
			"https://github.com/rancher/charts": "dev-v2.8",
		},
		"v2.7": {
			// rancher-charts
			"https://github.com/rancher/charts": "dev-v2.7",
		},
		"v2.6": {
			// rancher-charts
			"https://github.com/rancher/charts": "dev-v2.6",
		},
		"v2.5": {
			// system-chart
			"https://github.com/rancher/system-charts": "dev-v2.5",
			// rancher-charts
			"https://github.com/rancher/charts": "dev-v2.5",
		},
	}

	// map[version]map[url][branch]
	RancherPrimeManagerSystemChartsDEV = map[string]map[string]string{
		"v2.8": {
			// system-chart
			"https://github.com/rancher/system-charts": "dev-v2.8",
		},
		"v2.7": {
			// system-chart
			"https://github.com/rancher/system-charts": "dev-v2.7",
		},
		"v2.6": {
			// system-chart
			"https://github.com/rancher/system-charts": "dev-v2.6",
		},
		"v2.5": {
			// system-chart
			"https://github.com/rancher/system-charts": "dev-v2.5",
		},
	}

	// map[version]url
	KontainerDriverMetadataURLs = map[string]string{
		"v2.8": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.8/data.json",
		"v2.7": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.7/data.json",
		"v2.6": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.6/data.json",
		"v2.5": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.5/data.json",
	}

	// map[version]url
	KontainerDriverMetadataGCURLs = map[string]string{
		"v2.8": "https://charts.rancher.cn/kontainer-driver-metadata/release-v2.8/data.json",
		"v2.7": "https://charts.rancher.cn/kontainer-driver-metadata/release-v2.7/data.json",
		"v2.6": "https://charts.rancher.cn/kontainer-driver-metadata/release-v2.6/data.json",
		// The 2.5 KDM data is same with upstream
		"v2.5": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.5/data.json",
	}

	// map[version]url
	KontainerDriverMetadataURLsDEV = map[string]string{
		"v2.8": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.8/data.json",
		"v2.7": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.7/data.json",
		"v2.6": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.6/data.json",
		"v2.5": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.5/data.json",
	}

	// map[version]url
	KontainerDriverMetadataGCURLsDEV = map[string]string{
		"v2.8": "https://charts.rancher.cn/kontainer-driver-metadata/dev-v2.8/data.json",
		"v2.7": "https://charts.rancher.cn/kontainer-driver-metadata/dev-v2.7/data.json",
		"v2.6": "https://charts.rancher.cn/kontainer-driver-metadata/dev-v2.6/data.json",
		"v2.5": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.5/data.json",
	}
)

func addRPMCharts(v string, o *listgenerator.GeneratorOption, dev bool) {
	majorMinor := semver.MajorMinor(v)
	chartsMap := RancherPrimeManagerCharts
	if dev {
		chartsMap = RancherPrimeManagerChartsDEV
	}
	for url := range chartsMap[majorMinor] {
		o.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeDefault,
			Branch: chartsMap[majorMinor][url],
		}
	}
}

func addRPMSystemCharts(v string, o *listgenerator.GeneratorOption, dev bool) {
	majorMinor := semver.MajorMinor(v)
	systemChartsMap := RancherPrimeManagerSystemCharts
	if dev {
		systemChartsMap = RancherPrimeManagerSystemChartsDEV
	}
	for url := range systemChartsMap[majorMinor] {
		o.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeSystem,
			Branch: systemChartsMap[majorMinor][url],
		}
	}
}

func addRPMGCCharts(v string, o *listgenerator.GeneratorOption, dev bool) {
	majorMinor := semver.MajorMinor(v)
	chartsMap := RancherPrimeManagerGCCharts
	if dev {
		chartsMap = RancherPrimeManagerGCChartsDEV
	}
	for url := range chartsMap[majorMinor] {
		o.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeDefault,
			Branch: chartsMap[majorMinor][url],
		}
	}
}

func addRPMGCSystemCharts(v string, o *listgenerator.GeneratorOption, dev bool) {
	majorMinor := semver.MajorMinor(v)
	chartsMap := RancherPrimeManagerGCSystemCharts
	if dev {
		chartsMap = RancherPrimeManagerGCSystemChartsDEV
	}
	for url := range chartsMap[majorMinor] {
		o.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeSystem,
			Branch: chartsMap[majorMinor][url],
		}
	}
}

func addRancherPrimeManagerKontainerDriverMetadata(
	v string, o *listgenerator.GeneratorOption, dev bool,
) {
	majorMinor := semver.MajorMinor(v)
	urlMap := KontainerDriverMetadataURLs
	if dev {
		urlMap = KontainerDriverMetadataURLsDEV
	}
	url, ok := urlMap[majorMinor]
	if !ok {
		logrus.Warnf("KDM URL of version %q not found!", majorMinor)
		return
	}
	o.KDMURL = url
}

func addRancherPrimeManagerGCKontainerDriverMetadata(
	v string, o *listgenerator.GeneratorOption, dev bool,
) {
	urlMap := KontainerDriverMetadataURLs
	if dev {
		urlMap = KontainerDriverMetadataURLsDEV
	}
	if shouldUseGCKDM(v) {
		if dev {
			urlMap = KontainerDriverMetadataGCURLsDEV
		} else {
			urlMap = KontainerDriverMetadataGCURLs
		}
	}

	majorMinor := semver.MajorMinor(v)
	url, ok := urlMap[majorMinor]
	if !ok {
		logrus.Warnf("KDM URL of version %q not found!", majorMinor)
		return
	}
	o.KDMURL = url
}

func shouldUseGCKDM(version string) bool {
	// v2.8.5 and v2.9.0+ does not required to use GC KDM anymore
	if n, e := utils.SemverCompare(version, "v2.9.0"); e == nil && n >= 0 {
		return false
	} else if n, e := utils.SemverCompare(version, "v2.8.5"); e == nil && n >= 0 {
		return false
	}
	return true
}
