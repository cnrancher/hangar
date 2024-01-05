package kdmimages

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
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
	K3SImageURL      = "https://github.com/k3s-io/k3s/releases/download/%s/k3s-images.txt"

	// CN mirror URL (some versions were not mirrored to the CN mirror, so use the global GitHub instead)
	RKE2ImageLinuxCN   = "https://rancher-mirror.rancher.cn/rke2/%s/rke2-images-all.linux-amd64.txt"
	RKE2ImageWindowsCN = "https://rancher-mirror.rancher.cn/rke2/%s/rke2-images.windows-amd64.txt"
	K3SImageURLCN      = "https://rancher-mirror.rancher.cn/k3s/%s/k3s-images.txt"
)

// K3sRKE2Getter is the object to get RKE2 and K3s release & upgrade images
type K3sRKE2Getter struct {
	source             ClusterType
	rancherVersion     string
	minKubeVersion     string
	data               map[string]any
	insecureSkipVerify bool

	linuxImageSet   map[string]map[string]bool
	windowsImageSet map[string]map[string]bool
	versionSet      map[string]bool
}

func NewK3sRKE2Getter(
	source ClusterType,
	rancherVersion string,
	minKubeVersion string,
	data map[string]any,
	skipTLS bool,
) (*K3sRKE2Getter, error) {
	if source == "" || source == RKE {
		return nil, fmt.Errorf("invalid cluster type: %v", source)
	}
	if _, err := utils.EnsureSemverValid(rancherVersion); err != nil {
		return nil, err
	}
	if _, err := utils.EnsureSemverValid(minKubeVersion); err != nil {
		return nil, err
	}

	return &K3sRKE2Getter{
		source:             source,
		rancherVersion:     rancherVersion,
		minKubeVersion:     minKubeVersion,
		data:               data,
		insecureSkipVerify: skipTLS,

		linuxImageSet:   make(map[string]map[string]bool),
		windowsImageSet: make(map[string]map[string]bool),
		versionSet:      make(map[string]bool),
	}, nil
}

func (g *K3sRKE2Getter) Get(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	logrus.Infof("Fetching [%v] images.", g.source)
	releases, ok := g.data["releases"].([]any)
	if !ok {
		return fmt.Errorf("UpgradeGetter: failed to get 'releases' from data")
	}
	var compatibleVersions []string
	for _, release := range releases {
		releaseMap, ok := release.(map[string]any)
		if !ok {
			continue
		}

		kubeVersion, ok := releaseMap["version"].(string)
		if !ok || kubeVersion == "" {
			continue
		}

		if g.minKubeVersion != "" {
			// skip if kubeVersion is less than MinKubeVersion
			if !semver.IsValid(kubeVersion) {
				continue
			}
			if semver.Compare(kubeVersion, g.minKubeVersion) < 0 {
				continue
			}
		}

		if g.rancherVersion == "dev" {
			logrus.Debugf("[%s] adding compatible release: %s",
				g.source, kubeVersion)
			compatibleVersions = append(compatibleVersions, kubeVersion)
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
		if semver.Compare(g.rancherVersion, minVersion) < 0 {
			// Rancher version not equal to or less than \
			// minimum supported rancher version.
			continue
		}
		if semver.Compare(g.rancherVersion, maxVersion) > 0 {
			// Rancher version not equal to or greater than \
			// maximum supported rancher version.
			continue
		}

		logrus.Debugf("[%s] adding compatible release: %s",
			g.source, kubeVersion)
		compatibleVersions = append(compatibleVersions, kubeVersion)
	}

	if len(compatibleVersions) == 0 {
		logrus.Infof("skipping image generation since no compatible releases "+
			"were found for version: %s", g.rancherVersion)
		return nil
	}

	rs := fmt.Sprintf("[%s-release(rancher)]", g.source)
	us := fmt.Sprintf("[%s-upgrade(rancher)]", g.source)
	for _, version := range compatibleVersions {
		g.versionSet[version] = true

		// Add upgrade images.
		upgradeImage := fmt.Sprintf("rancher/%s-upgrade:%s",
			g.source, strings.ReplaceAll(version, "+", "-"))
		if g.linuxImageSet[upgradeImage] == nil {
			g.linuxImageSet[upgradeImage] = make(map[string]bool)
		}
		g.linuxImageSet[upgradeImage][us] = true

		// Add system-agent-installer images.
		systemAgentInstallerImage := fmt.Sprintf(
			"%s%s:%s", "rancher/system-agent-installer-",
			g.source, strings.ReplaceAll(version, "+", "-"))
		if g.linuxImageSet[systemAgentInstallerImage] == nil {
			g.linuxImageSet[systemAgentInstallerImage] = make(map[string]bool)
		}
		g.linuxImageSet[systemAgentInstallerImage][rs] = true

		// Get linux images from GitHub Release.
		linuxImages, err := g.getLinuxExternalList(ctx, version)
		if err != nil {
			logrus.Errorf("Could not download linux images for %s [%s]: %v",
				g.source, version, err)
			return err
		}
		for _, img := range linuxImages {
			if g.linuxImageSet[img] == nil {
				g.linuxImageSet[img] = make(map[string]bool)
			}
			g.linuxImageSet[img][rs] = true
		}

		// Get windows images from GitHub Release.
		windowsImages, err := g.getWindowsExternalList(ctx, version)
		if err != nil {
			logrus.Errorf("Could not download windows images for %s [%s]: %v",
				g.source, version, err)
			return err
		}
		for _, img := range windowsImages {
			if g.windowsImageSet[img] == nil {
				g.windowsImageSet[img] = make(map[string]bool)
			}
			g.windowsImageSet[img][rs] = true
		}
	}

	return nil
}

func (g *K3sRKE2Getter) getLinuxExternalList(ctx context.Context, release string) ([]string, error) {
	var link string
	switch g.source {
	case RKE2:
		link = fmt.Sprintf(RKE2ImageLinux, release)
	case K3S:
		link = fmt.Sprintf(K3SImageURL, release)
	default:
		return nil, fmt.Errorf("invalid image source: %v", g.source)
	}
	return getImageListFromURL(ctx, g.insecureSkipVerify, link)
}

func (g *K3sRKE2Getter) getWindowsExternalList(ctx context.Context, release string) ([]string, error) {
	var link string
	switch g.source {
	case RKE2:
		link = fmt.Sprintf(RKE2ImageWindows, release)
	case K3S:
		// K3s does not support Windows.
		return []string{}, nil
	default:
		return nil, fmt.Errorf("invalid image source: %v", g.source)
	}
	return getImageListFromURL(ctx, g.insecureSkipVerify, link)
}

func getImageListFromURL(ctx context.Context, tlsVerify bool, link string) ([]string, error) {
	logrus.Infof("Get images from %q", link)
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !tlsVerify},
			Proxy:           http.ProxyFromEnvironment,
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
		l = strings.TrimPrefix(l, "docker.io/")
		list = append(list, l)
	}
	return list, nil
}

func (g *K3sRKE2Getter) LinuxImageSet() map[string]map[string]bool {
	return g.linuxImageSet
}

func (g *K3sRKE2Getter) WindowsImageSet() map[string]map[string]bool {
	return g.windowsImageSet
}

func (g *K3sRKE2Getter) VersionSet() map[string]bool {
	return g.versionSet
}

func (g *K3sRKE2Getter) Source() ClusterType {
	return g.source
}
