package kdmimages

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type ReleaseImages struct {
	Source string
	Data   map[string]interface{}
}

func (r *ReleaseImages) GetImages() ([]string, error) {
	if r.Data == nil {
		return nil, fmt.Errorf("GetImages: Data is nil")
	}
	if r.Source == "" || (r.Source != RKE2 && r.Source != K3S) {
		return nil, fmt.Errorf("GetImages: invalid source %q", r.Source)
	}
	logrus.Infof("generating KDM release images...")
	versions, err := r.GetVersions()
	if err != nil {
		return nil, err
	}
	var images = make([]string, 0, len(versions))
	for i := range versions {
		image := fmt.Sprintf(
			"rancher/system-agent-installer-%s:%s", r.Source, versions[i])
		images = append(images, image)
	}
	logrus.Infof("finished generating KDM release images")
	return images, nil
}

func (r *ReleaseImages) GetVersions() ([]string, error) {
	var versions []string
	releases, ok := r.Data["releases"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("GetVersions: failed to get releases from data")
	}
	for _, v := range releases {
		releaseMap, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		_, ok = releaseMap["serverArgs"].(map[string]interface{})
		if !ok {
			continue
		}
		version, ok := releaseMap["version"].(string)
		if !ok {
			continue
		}
		version = strings.ReplaceAll(version, "+", "-")
		versions = append(versions, version)
	}
	return versions, nil
}
