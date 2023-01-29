package chartimages

import (
	"fmt"
	"os"
	"testing"

	"github.com/cnrancher/image-tools/pkg/utils"
	"github.com/sirupsen/logrus"
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
	for source := range chart.ImageSet {
		for image := range chart.ImageSet[source] {
			l := fmt.Sprintf("%s [%s]", image, source)
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
	for source := range chart.ImageSet {
		for image := range chart.ImageSet[source] {
			l := fmt.Sprintf("%s [%s]", image, source)
			utils.AppendFileLine("test/system-charts-linux.txt", l)
		}
	}
}
