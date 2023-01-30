package kdmimages_test

import (
	"os"
	"testing"

	"github.com/cnrancher/image-tools/pkg/rancher/kdmimages"
	"github.com/cnrancher/image-tools/pkg/utils"
	"github.com/rancher/rke/types/kdm"
)

func Test_Upgrade_GetImages(t *testing.T) {
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
	ug := kdmimages.UpgradeImages{
		Source:         kdmimages.RKE2,
		RancherVersion: "v2.7.0",
		MinKubeVersion: "v1.21.0",
		Data:           data.RKE2,
	}
	images, err := ug.GetImages()
	if err != nil {
		t.Error(err)
	}
	utils.SaveSlice("test/rke2-upgrade-images.txt", images)
	ug.Source = kdmimages.K3S
	ug.Data = data.K3S
	images, err = ug.GetImages()
	if err != nil {
		t.Error(err)
	}
	utils.SaveSlice("test/k3s-upgrade-images.txt", images)
}
