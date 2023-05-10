package chartimages

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func Test_fetchChartsFromPath(t *testing.T) {
	chart := Chart{
		RancherVersion: "v2.7.0",
		OS:             Linux,
		Type:           RepoTypeDefault,
		Path:           "test/pandaria-catalog",
		URL:            "",
		ImageSet:       make(map[string]map[string]bool),
	}
	err := chart.fetchChartsFromPath()
	if os.IsNotExist(err) {
		// skip if not exists
		logrus.Warnf("%q does not exists", chart.Path)
		return
	}
	if err != nil {
		t.Error(err)
	}
	utils.DeleteIfExist("test/pandaria-catalog-linux.txt")
	utils.AppendFileLine("test/pandaria-catalog-linux.txt", "# IMAGE SOURCE")
	for image := range chart.ImageSet {
		for source := range chart.ImageSet[image] {
			l := fmt.Sprintf("%s %s", image, source)
			utils.AppendFileLine("test/pandaria-catalog-linux.txt", l)
		}
	}

	chart = Chart{
		RancherVersion: "v2.7.0",
		OS:             Linux,
		Type:           RepoTypeSystem,
		Path:           "test/system-charts",
		URL:            "",
		ImageSet:       make(map[string]map[string]bool),
	}
	err = chart.fetchChartsFromPath()
	if os.IsNotExist(err) {
		// skip if not exists
		logrus.Warnf("%q does not exists", chart.Path)
		return
	}
	if err != nil {
		t.Error(err)
	}
	utils.DeleteIfExist("test/system-charts-linux.txt")
	utils.AppendFileLine("test/system-charts-linux.txt", "# IMAGE SOURCE")
	for image := range chart.ImageSet {
		for source := range chart.ImageSet[image] {
			l := fmt.Sprintf("%s %s", image, source)
			utils.AppendFileLine("test/system-charts-linux.txt", l)
		}
	}
}

func Test_BuildOrGetIndex(t *testing.T) {
	index, err := BuildOrGetIndex("test/system-charts")
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Error(err)
		return
	}

	assert.NotEqual(t, len(index.Entries), 0)
	for name, versions := range index.Entries {
		assert.NotEqual(t, name, "")
		for _, v := range versions {
			assert.NotEqual(t, v.APIVersion, "")
			assert.NotEqual(t, v.AppVersion, "",
				"chart %v:%v appVersion is empty", name, v.Version)
			assert.NotEqual(t, v.Version, "")
			assert.NotEqual(t, v.Name, "")
		}
	}
	versions := index.Entries["rancher-logging"]
	if versions == nil {
		t.Error("versions of rancher-logging is nil")
		return
	}
	t.Logf("Name: %s", versions[0].Name)
	for _, v := range versions {
		t.Logf("%s", v.Version)
	}
	maxVersion := versions[0]
	if maxVersion.Version != "0.3.1001" {
		t.Error("failed: max version is ", maxVersion.Version)
	}
}

func Test_PickImagesFromValuesMap(t *testing.T) {
	imageSet := map[string]map[string]bool{}
	r, err := os.Open(
		"test/pandaria-catalog/charts/rancher-macvlan/0.8.2/values.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Error(err)
	}
	value := map[interface{}]interface{}{}
	err = decodeYAMLFile(r, value)
	if err != nil {
		t.Error(err)
	}

	err = PickImagesFromValuesMap(imageSet, value, "test", Linux)
	if err != nil {
		t.Error(err)
	}
	counter := 0
	for image := range imageSet {
		t.Logf("%v\n", image)
		counter++
	}
	assert.NotZero(t, counter)
}

func Test_DecodeValuesInDir(t *testing.T) {
	values, err := DecodeValuesInDir(
		"test/rancher-charts/charts/epinio/102.0.1+up1.6.2")
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Error(err)
		return
	}
	imageSet := map[string]map[string]bool{}
	for _, value := range values {
		PickImagesFromValuesMap(imageSet, value, "test", Linux)
	}
	var flag = false
	for image := range imageSet {
		if strings.Contains(image, "epinio") {
			t.Logf("%s\n", image)
			flag = true
		}
	}
	if !flag {
		t.Error("failed")
	}
}

func Test_DecodeValuesInTgz(t *testing.T) {
	values, err := DecodeValuesInTgz(
		"test/rancher-charts/assets/epinio/epinio-102.0.1+up1.6.2.tgz")
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Error(err)
		return
	}
	imageSet := map[string]map[string]bool{}
	for _, value := range values {
		PickImagesFromValuesMap(imageSet, value, "test", Linux)
	}
	var flag = false
	for image := range imageSet {
		if strings.Contains(image, "epinio") {
			t.Logf("%s\n", image)
			flag = true
		}
	}
	if !flag {
		t.Error("failed")
	}
}

func Test_fetchChartsFromPath_RancherCharts(t *testing.T) {
	chart := Chart{
		RancherVersion: "v2.7.0",
		OS:             Linux,
		Type:           RepoTypeDefault,
		Path:           "test/rancher-charts",
		URL:            "",
		ImageSet:       make(map[string]map[string]bool),
	}
	err := chart.fetchChartsFromPath()
	if os.IsNotExist(err) {
		// skip if not exists
		logrus.Warnf("%q does not exists", chart.Path)
		return
	}
	if err != nil {
		t.Error(err)
	}
	utils.DeleteIfExist("test/rancher-charts-linux.txt")
	utils.AppendFileLine("test/rancher-charts-linux.txt", "# IMAGE SOURCE")
	for image := range chart.ImageSet {
		for source := range chart.ImageSet[image] {
			l := fmt.Sprintf("%s %s", image, source)
			utils.AppendFileLine("test/rancher-charts-linux.txt", l)
		}
	}
}
