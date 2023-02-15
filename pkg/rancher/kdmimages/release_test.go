package kdmimages_test

import (
	"embed"
	"os"
	"testing"

	"github.com/cnrancher/hangar/pkg/rancher/kdmimages"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/rancher/rke/types/kdm"
	"github.com/sirupsen/logrus"
)

//go:embed test/*.json
var testFs embed.FS

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func Test_GetVersions(t *testing.T) {
	b, err := testFs.ReadFile("test/data.json")
	if err != nil {
		if os.IsNotExist(err) {
			t.Log("skip test")
		}
		t.Error(err)
		return
	}
	data, err := kdm.FromData(b)
	if err != nil {
		t.Error(err)
		return
	}
	// Get RKE2 images
	rv := kdmimages.ReleaseImages{
		Source: kdmimages.RKE2,
		Data:   data.RKE2,
	}
	versions, err := rv.GetVersions()
	if err != nil {
		t.Error(err)
		return
	}
	utils.SaveSlice("test/rke2-release-versions.txt", versions)
	// Get K3S images
	rv.Source = kdmimages.K3S
	rv.Data = data.K3S
	versions, err = rv.GetVersions()
	if err != nil {
		t.Error(err)
		return
	}
	utils.SaveSlice("test/k3s-release-versions.txt", versions)
}

func Test_GetImages(t *testing.T) {
	b, err := testFs.ReadFile("test/data.json")
	if err != nil {
		if os.IsNotExist(err) {
			t.Log("skip test")
			return
		}
		t.Error(err)
		return
	}
	data, err := kdm.FromData(b)
	if err != nil {
		t.Error(err)
		return
	}
	// Get RKE2 images
	rv := kdmimages.ReleaseImages{
		Source: kdmimages.RKE2,
		Data:   data.RKE2,
	}
	images, err := rv.GetImages()
	if err != nil {
		t.Error(err)
		return
	}
	utils.SaveSlice("test/rke2-release-images.txt", images)
	// Get K3S images
	rv.Source = kdmimages.K3S
	rv.Data = data.K3S
	images, err = rv.GetImages()
	if err != nil {
		t.Error(err)
		return
	}
	utils.SaveSlice("test/k3s-release-images.txt", images)
}
