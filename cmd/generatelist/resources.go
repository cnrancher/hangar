package generatelist

import (
	"github.com/cnrancher/image-tools/pkg/rancher/chartimages"
	"github.com/cnrancher/image-tools/pkg/rancher/listgenerator"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

var (
	// map[version]map[url][branch]
	RPM_GC_CHARTS = map[string]map[string]string{
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
	RPM_GC_SYSTEM_CHARTS = map[string]map[string]string{
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
	RPM_GC_CHARTS_DEV = map[string]map[string]string{
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
	RPM_GC_SYSTEM_CHARTS_DEV = map[string]map[string]string{
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
	RPM_CHARTS = map[string]map[string]string{
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
	RPM_SYSTEM_CHARTS = map[string]map[string]string{
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
	RPM_CHARTS_DEV = map[string]map[string]string{
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
	RPM_SYSTEM_CHARTS_DEV = map[string]map[string]string{
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
	KDM_URLS = map[string]string{
		"v2.7": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.7/data.json",
		"v2.6": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.6/data.json",
		"v2.5": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.5/data.json",
	}

	// map[version]url
	KDM_GC_URLS = map[string]string{
		"v2.7": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.7/data.json",
		"v2.6": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.6/data.json",
		"v2.5": "https://releases.rancher.com/kontainer-driver-metadata/release-v2.5/data.json",
	}

	// map[version]url
	KDM_URLS_DEV = map[string]string{
		"v2.7": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.7/data.json",
		"v2.6": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.6/data.json",
		"v2.5": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.5/data.json",
	}

	// map[version]url
	KDM_GC_URLS_DEV = map[string]string{
		"v2.7": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.7/data.json",
		"v2.6": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.6/data.json",
		"v2.5": "https://releases.rancher.com/kontainer-driver-metadata/dev-v2.5/data.json",
	}
)

func AddRPMCharts(v string, g *listgenerator.Generator, dev bool) {
	majorMinor := semver.MajorMinor(v)
	chartsMap := RPM_CHARTS
	if dev {
		chartsMap = RPM_CHARTS_DEV
	}
	for url := range chartsMap[majorMinor] {
		g.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeDefault,
			Branch: chartsMap[majorMinor][url],
		}
	}
	systemChartsMap := RPM_SYSTEM_CHARTS
	if dev {
		systemChartsMap = RPM_SYSTEM_CHARTS_DEV
	}
	for url := range systemChartsMap[majorMinor] {
		g.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeSystem,
			Branch: systemChartsMap[majorMinor][url],
		}
	}
}

func AddRPMGCCharts(v string, g *listgenerator.Generator, dev bool) {
	majorMinor := semver.MajorMinor(v)
	chartsMap := RPM_GC_CHARTS
	if dev {
		chartsMap = RPM_GC_CHARTS_DEV
	}
	for url := range chartsMap[majorMinor] {
		g.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeDefault,
			Branch: chartsMap[majorMinor][url],
		}
	}
	chartsMap = RPM_GC_SYSTEM_CHARTS
	if dev {
		chartsMap = RPM_GC_SYSTEM_CHARTS_DEV
	}
	for url := range chartsMap[majorMinor] {
		g.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeSystem,
			Branch: chartsMap[majorMinor][url],
		}
	}
}

func AddRPM_KDM(v string, g *listgenerator.Generator, dev bool) {
	majorMinor := semver.MajorMinor(v)
	urlMap := KDM_URLS
	if dev {
		urlMap = KDM_URLS_DEV
	}
	url, ok := urlMap[majorMinor]
	if !ok {
		logrus.Warnf("KDM URL of version %q not found!", majorMinor)
		return
	}
	g.KDMURL = url
}

func AddRPM_GC_KDM(v string, g *listgenerator.Generator, dev bool) {
	majorMinor := semver.MajorMinor(v)
	urlMap := KDM_GC_URLS
	if dev {
		urlMap = KDM_GC_URLS_DEV
	}
	url, ok := urlMap[majorMinor]
	if !ok {
		logrus.Warnf("KDM URL of version %q not found!", majorMinor)
		return
	}
	g.KDMURL = url
}
