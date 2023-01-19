package listgenerator

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/rancher/rke/types/kdm"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func Test_DataJson(t *testing.T) {
	b, err := os.ReadFile("test/rancher-data.json")
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Error(err)
	}

	data, err := kdm.FromData(b)
	if err != nil {
		t.Error(err)
		return
	}
	kdmData := kdm.Data{
		K3S:  data.K3S,
		RKE2: data.RKE2,
	}
	b, err = json.MarshalIndent(kdmData, "", "  ")
	if err != nil {
		t.Error(err)
		return
	}
	err = os.WriteFile("test/data.json", b, 0644)
	if err != nil {
		t.Error(err)
	}
}

func Test_generateFromKDMPath(t *testing.T) {
	g := Generator{
		RancherVersion: "v2.7.0",
		KDMPath:        "test/data.json",
	}
	g.init()
	err := g.generateFromKDMPath()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Error(err)
	}
	for source, imageMap := range g.GeneratedLinuxImages {
		for k := range imageMap {
			t.Logf("[%v] %s", source, k)
		}
	}
}

func Test_generateFromKDMURL(t *testing.T) {
	g := Generator{
		RancherVersion: "v2.7.0",
		KDMURL:         "",
	}
	g.init()
	err := g.generateFromKDMURL()
	if err != nil {
		t.Error(err)
	}
	for source, imageMap := range g.GeneratedLinuxImages {
		for k := range imageMap {
			t.Logf("[%v] %s", source, k)
		}
	}
}
