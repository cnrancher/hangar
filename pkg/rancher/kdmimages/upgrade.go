package kdmimages

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

const (
	// The "images-all" file is only provided for RKE2 amd64 images. This may be subject to change.
	RKE2ImageLinux   = "https://github.com/rancher/rke2/releases/download/%s/rke2-images-all.linux-amd64.txt"
	RKE2ImageWindows = "https://github.com/rancher/rke2/releases/download/%s/rke2-images.windows-amd64.txt"

	K3SImageURL = "https://github.com/k3s-io/k3s/releases/download/%s/k3s-images.txt"

	K3S  = "k3s"
	RKE2 = "rke2"
)

// UpgradeImages generates external image list from KDM RKE2/K3S data
type UpgradeImages struct {
	Source             string
	RancherVersion     string
	MinKubeVersion     string
	InsecureSkipVerify bool
	Data               map[string]interface{}
}

func (g *UpgradeImages) GetImages(ctx context.Context) ([]string, error) {
	if g.Source != K3S && g.Source != RKE2 {
		return nil, fmt.Errorf("invalid source provided: %v", g.Source)
	}

	logrus.Infof("Generating %s upgrade images...", g.Source)
	releases, ok := g.Data["releases"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to get 'releases' from data")
	}
	var compatibleReleases []string
	for _, release := range releases {
		releaseMap, ok := release.(map[string]interface{})
		if !ok {
			continue
		}

		kubeVersion, ok := releaseMap["version"].(string)
		if !ok || kubeVersion == "" {
			continue
		}

		if g.MinKubeVersion != "" {
			// skip if kubeVersion is less than MinKubeVersion
			if !semver.IsValid(kubeVersion) {
				continue
			}
			if semver.Compare(kubeVersion, g.MinKubeVersion) < 0 {
				continue
			}
		}

		if g.RancherVersion == "dev" {
			logrus.Debugf("[%s] adding compatible release: %s",
				g.Source, kubeVersion)
			compatibleReleases = append(compatibleReleases, kubeVersion)
			continue
		}
		maxVersion, ok := releaseMap["maxChannelServerVersion"].(string)
		if !ok || !semver.IsValid(maxVersion) {
			continue
		}
		minVersion, ok := releaseMap["minChannelServerVersion"].(string)
		if !ok || !semver.IsValid(minVersion) {
			continue
		}
		if semver.Compare(g.RancherVersion, minVersion) < 0 {
			// Rancher version not equal to or less than \
			// minimum supported rancher version.
			continue
		}
		if semver.Compare(g.RancherVersion, maxVersion) > 0 {
			// Rancher version not equal to or greater than \
			// maximum supported rancher version.
			continue
		}

		logrus.Debugf("[%s] adding compatible release: %s",
			g.Source, kubeVersion)
		compatibleReleases = append(compatibleReleases, kubeVersion)
	}

	if len(compatibleReleases) == 0 {
		logrus.Infof("skipping image generation since no compatible releases "+
			"were found for version: %s", g.RancherVersion)
		return nil, nil
	}

	// use map to deduplication
	externalImagesMap := make(map[string]bool)
	for _, release := range compatibleReleases {
		// Replace '+' to '-'
		upgradeImage := fmt.Sprintf("rancher/%s-upgrade:%s",
			g.Source, strings.ReplaceAll(release, "+", "-"))
		externalImagesMap[upgradeImage] = true
		systemAgentInstallerImage := fmt.Sprintf(
			"%s%s:%s", "rancher/system-agent-installer-",
			g.Source, strings.ReplaceAll(release, "+", "-"))
		externalImagesMap[systemAgentInstallerImage] = true

		images, err := g.getExternalList(ctx, release)
		if err != nil {
			logrus.Errorf(
				"could not find supporting images for %s release [%s]: %v",
				g.Source, release, err)
			return nil, err
		}

		for _, name := range images {
			name = strings.TrimPrefix(name, "docker.io/")
			externalImagesMap[name] = true
		}
	}

	var externalImages []string
	for imageName := range externalImagesMap {
		externalImages = append(externalImages, imageName)
	}
	sort.Strings(externalImages)
	logrus.Infof("Finished generating %s upgrade images", g.Source)

	return externalImages, nil
}

func (g *UpgradeImages) getExternalList(ctx context.Context, release string) ([]string, error) {
	switch g.Source {
	case RKE2:
		linuxImages, err := getImageListFromURL(
			ctx, g.InsecureSkipVerify, fmt.Sprintf(RKE2ImageLinux, release))
		if err != nil {
			return nil, err
		}
		// windowsImages, err := getImageListFromURL(
		// 	fmt.Sprintf(RKE2ImageWindows, release))
		// if err != nil {
		// 	return nil, err
		// }
		// return append(linuxImages, windowsImages...), nil
		return linuxImages, nil
	case K3S:
		return getImageListFromURL(
			ctx, g.InsecureSkipVerify, fmt.Sprintf(K3SImageURL, release))
	default:
		return nil, fmt.Errorf("invalid source provided: %s", g.Source)
	}
}

func getImageListFromURL(ctx context.Context, tlsVerify bool, link string) ([]string, error) {
	logrus.Infof("Getting image list from %q", link)
	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !tlsVerify},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, fmt.Errorf("getImageListFromURL: %w", err)
	}
	resp, err := utils.HTTPClientDoWithRetry(ctx, client, req)
	if err != nil {
		return nil, fmt.Errorf("getImageListFromURL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get url %q: %v", link, resp.Status)
	}

	list := []string{}
	sc := bufio.NewScanner(resp.Body)
	sc.Split(bufio.ScanLines)
	for sc.Scan() {
		l := sc.Text()
		if l == "" {
			continue
		}
		list = append(list, l)
	}
	return list, nil
}
