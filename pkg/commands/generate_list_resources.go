package commands

import (
	"fmt"

	"github.com/cnrancher/hangar/pkg/rancher/chartimages"
	"github.com/cnrancher/hangar/pkg/rancher/listgenerator"
	"github.com/cnrancher/hangar/pkg/utils"
	"golang.org/x/mod/semver"
)

const (
	// Global Rancher Prime
	RancherPrimeChartsRepo       = "https://github.com/rancher/charts"
	RancherPrimeSystemChartsRepo = "https://github.com/rancher/system-charts"
	KontainerDriverMetadataURL   = "https://releases.rancher.com/kontainer-driver-metadata"

	// Rancher Prime GC
	RancherPrimeGCChartsRepo       = "https://github.com/cnrancher/pandaria-catalog"
	RancherPrimeGCSystemChartsRepo = "https://github.com/cnrancher/system-charts"
	KontainerDriverMetadataGCURL   = "https://charts.rancher.cn/kontainer-driver-metadata"
)

func addRancherPrimeCharts(v string, o *listgenerator.GeneratorOption, dev bool) {
	majorMinor := semver.MajorMinor(v)
	var branch string
	if dev {
		branch = fmt.Sprintf("dev-%v", majorMinor)
	} else {
		branch = fmt.Sprintf("release-%v", majorMinor)
	}

	o.ChartURLs[RancherPrimeChartsRepo] = struct {
		Type   chartimages.ChartRepoType
		Branch string
	}{
		Type:   chartimages.RepoTypeDefault,
		Branch: branch,
	}
}

func addRancherPrimeSystemCharts(
	v string, o *listgenerator.GeneratorOption, dev bool,
) {
	if semver.Compare(v, "v2.11.0-0") >= 0 {
		// SystemChart was removed on v2.11
		return
	}
	majorMinor := semver.MajorMinor(v)
	var branch string
	if dev {
		branch = fmt.Sprintf("dev-%v", majorMinor)
	} else {
		branch = fmt.Sprintf("release-%v", majorMinor)
	}

	o.ChartURLs[RancherPrimeSystemChartsRepo] = struct {
		Type   chartimages.ChartRepoType
		Branch string
	}{
		Type:   chartimages.RepoTypeSystem,
		Branch: branch,
	}
}

func addRancherPrimeGCCharts(
	v string, o *listgenerator.GeneratorOption, dev bool,
) {
	majorMinor := semver.MajorMinor(v)
	var branch string
	if dev {
		branch = fmt.Sprintf("dev/%v", majorMinor)
	} else {
		branch = fmt.Sprintf("release/%v", majorMinor)
	}

	o.ChartURLs[RancherPrimeGCChartsRepo] = struct {
		Type   chartimages.ChartRepoType
		Branch string
	}{
		Type:   chartimages.RepoTypeDefault,
		Branch: branch,
	}
}

func addRancherPrimeGCSystemCharts(v string, o *listgenerator.GeneratorOption, dev bool) {
	majorMinor := semver.MajorMinor(v)
	var url string
	var branch string

	if semver.Compare(v, "v2.11.0-0") >= 0 {
		// SystemChart was removed on v2.11
		return
	}

	// GC starts use global system-charts from v2.9
	if semver.Compare(v, "v2.9.0") >= 0 {
		url = RancherPrimeSystemChartsRepo
		if dev {
			branch = fmt.Sprintf("dev-%v", majorMinor)
		} else {
			branch = fmt.Sprintf("release-%v", majorMinor)
		}
	} else {
		url = RancherPrimeGCSystemChartsRepo
		if dev {
			branch = fmt.Sprintf("dev-%v", majorMinor)
		} else {
			branch = fmt.Sprintf("release-%v-ent", majorMinor)
		}
	}

	o.ChartURLs[url] = struct {
		Type   chartimages.ChartRepoType
		Branch string
	}{
		Type:   chartimages.RepoTypeSystem,
		Branch: branch,
	}
}

func addRancherPrimeKontainerDriverMetadata(
	v string, o *listgenerator.GeneratorOption, dev bool,
) {
	majorMinor := semver.MajorMinor(v)
	var branch string
	if dev {
		branch = fmt.Sprintf("dev-%v", majorMinor)
	} else {
		branch = fmt.Sprintf("release-%v", majorMinor)
	}
	o.KDMURL = fmt.Sprintf("%v/%v/data.json", KontainerDriverMetadataURL, branch)
}

func addRancherPrimeManagerGCKontainerDriverMetadata(
	v string, o *listgenerator.GeneratorOption, dev bool,
) {
	majorMinor := semver.MajorMinor(v)
	var branch string
	if dev {
		branch = fmt.Sprintf("dev-%v", majorMinor)
	} else {
		branch = fmt.Sprintf("release-%v", majorMinor)
	}
	baseURL := KontainerDriverMetadataURL
	if shouldUseGCKDM(v) {
		baseURL = KontainerDriverMetadataGCURL
	}
	o.KDMURL = fmt.Sprintf("%v/%v/data.json", baseURL, branch)
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
