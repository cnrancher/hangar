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
)

func AddRPMCharts(v string, g *listgenerator.Generator) {
	majorMinor := semver.MajorMinor(v)
	for url := range RPM_CHARTS[majorMinor] {
		g.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeDefault,
			Branch: RPM_CHARTS[majorMinor][url],
		}
	}
	for url := range RPM_SYSTEM_CHARTS[majorMinor] {
		g.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeSystem,
			Branch: RPM_SYSTEM_CHARTS[majorMinor][url],
		}
	}
}

func AddRPMGCCharts(v string, g *listgenerator.Generator) {
	majorMinor := semver.MajorMinor(v)
	for url := range RPM_GC_CHARTS[majorMinor] {
		g.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeDefault,
			Branch: RPM_CHARTS[majorMinor][url],
		}
	}
	for url := range RPM_GC_SYSTEM_CHARTS[majorMinor] {
		g.ChartURLs[url] = struct {
			Type   chartimages.ChartRepoType
			Branch string
		}{
			Type:   chartimages.RepoTypeSystem,
			Branch: RPM_SYSTEM_CHARTS[majorMinor][url],
		}
	}
}

func AddRPM_KDM(v string, g *listgenerator.Generator) {
	majorMinor := semver.MajorMinor(v)
	url, ok := KDM_URLS[majorMinor]
	if !ok {
		logrus.Warnf("KDM URL of version %q not found!", majorMinor)
		return
	}
	g.KDMURL = url
}

func AddRPM_GC_KDM(v string, g *listgenerator.Generator) {
	majorMinor := semver.MajorMinor(v)
	url, ok := KDM_GC_URLS[majorMinor]
	if !ok {
		logrus.Warnf("KDM URL of version %q not found!", majorMinor)
		return
	}
	g.KDMURL = url
}
