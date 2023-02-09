package listgenerator

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/image-tools/pkg/rancher/chartimages"
	"github.com/cnrancher/image-tools/pkg/rancher/kdmimages"
	"github.com/cnrancher/image-tools/pkg/utils"
	u "github.com/cnrancher/image-tools/pkg/utils"
	"github.com/rancher/rke/types/kdm"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

// Generator is a generator to generate image list from charts, KDM data, etc.
type Generator struct {
	RancherVersion string // rancher version, should be va.b.c
	MinKubeVersion string // minimum kube verision, should be va.b.c

	ChartsPaths map[string]chartimages.ChartRepoType // map[url]type
	ChartURLs   map[string]struct {
		Type   chartimages.ChartRepoType
		Branch string
	}

	KDMPath string // the path of KDM data.json file
	KDMURL  string // the remote URL of KDM data.json

	WindowsImageArguments []string
	LinuxImageArguments   []string

	// generated images, map[image]map[source]true
	GeneratedLinuxImages   map[string]map[string]bool
	GeneratedWindowsImages map[string]map[string]bool
}

func (g *Generator) init() {
	if g.GeneratedLinuxImages == nil {
		g.GeneratedLinuxImages = make(map[string]map[string]bool)
	}
	if g.GeneratedWindowsImages == nil {
		g.GeneratedWindowsImages = make(map[string]map[string]bool)
	}
}

func (g *Generator) selfCheck() error {
	if g.RancherVersion == "" {
		return fmt.Errorf("RancherVersion is empty")
	}
	if !strings.HasPrefix(g.RancherVersion, "v") {
		g.RancherVersion = "v" + g.RancherVersion
	}
	if !semver.IsValid(g.RancherVersion) {
		return fmt.Errorf("%q is not a valid Rancher version", g.RancherVersion)
	}
	if g.ChartURLs == nil && g.ChartsPaths == nil &&
		g.KDMPath == "" && g.KDMURL == "" {
		return fmt.Errorf("no input source provided")
	}

	return nil
}

func (g *Generator) Generate() error {
	if err := g.selfCheck(); err != nil {
		return err
	}
	g.init()

	if err := g.generateFromChartPaths(); err != nil {
		return err
	}

	if err := g.generateFromChartURLs(); err != nil {
		return err
	}

	if err := g.generateFromKDMPath(); err != nil {
		return err
	}

	if err := g.generateFromKDMURL(); err != nil {
		return err
	}

	if err := g.handleImageArguments(); err != nil {
		return err
	}

	return nil
}

func (g *Generator) generateFromChartPaths() error {
	if g.ChartsPaths == nil || len(g.ChartsPaths) == 0 {
		return nil
	}
	for path := range g.ChartsPaths {
		c := chartimages.Chart{
			RancherVersion: g.RancherVersion,
			OS:             chartimages.Linux,
			Type:           g.ChartsPaths[path],
			Path:           path,
		}
		if err := c.FetchImages(); err != nil {
			return err
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.GeneratedLinuxImages, image, source)
			}
		}
		// fetch windows images
		c.OS = chartimages.Windows
		c.ImageSet = make(map[string]map[string]bool)
		if err := c.FetchImages(); err != nil {
			return err
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.GeneratedWindowsImages, image, source)
			}
		}
	}
	return nil
}

func (g *Generator) generateFromChartURLs() error {
	if g.ChartURLs == nil || len(g.ChartURLs) == 0 {
		return nil
	}
	for url := range g.ChartURLs {
		c := chartimages.Chart{
			RancherVersion: g.RancherVersion,
			OS:             chartimages.Linux,
			Type:           g.ChartURLs[url].Type,
			Branch:         g.ChartURLs[url].Branch,
			URL:            url,
		}
		if err := c.FetchImages(); err != nil {
			return err
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.GeneratedLinuxImages, image, source)
			}
		}
		// fetch windows images
		c.OS = chartimages.Windows
		c.ImageSet = make(map[string]map[string]bool)
		if err := c.FetchImages(); err != nil {
			return err
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.GeneratedWindowsImages, image, source)
			}
		}
		// Delete cloned chart path after generated images
		baseDir := strings.Split(c.Path, string(os.PathSeparator))[0]
		logrus.Debugf("Delete %q", baseDir)
		if err := u.DeleteIfExist(baseDir); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) generateFromKDMPath() error {
	if g.KDMPath == "" {
		return nil
	}
	b, err := os.ReadFile(g.KDMPath)
	if err != nil {
		return err
	}
	return g.generateFromKDMData(b)
}

func (g *Generator) generateFromKDMURL() error {
	if g.KDMURL == "" {
		return nil
	}
	logrus.Infof("Get KDM data from URL: %q", g.KDMURL)
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(g.KDMURL)
	if err != nil {
		// re-try get data from URL
		logrus.Warnf("Failed to get KDM data from url: %v, retrying...", err)
		resp, err = client.Get(g.KDMURL)
		if err != nil {
			return fmt.Errorf("generateFromKDMURL: http.Get: %w", err)
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("generateFromKDMURL: get url [%q]: %v",
			g.KDMURL, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("generateFromKDMURL: io.ReadAll: %w", err)
	}
	return g.generateFromKDMData(b)
}

func (g *Generator) generateFromKDMData(b []byte) error {
	data, err := kdm.FromData(b)
	if err != nil {
		return fmt.Errorf("generateFromKDMData: %w", err)
	}
	// get release images
	r := kdmimages.ReleaseImages{
		Source: kdmimages.K3S,
		Data:   data.K3S,
	}
	k3sReleaseImages, err := r.GetImages()
	if err != nil {
		return fmt.Errorf("generateFromKDMData: %w", err)
	}
	for _, image := range k3sReleaseImages {
		if g.GeneratedLinuxImages[image] == nil {
			g.GeneratedLinuxImages[image] = make(map[string]bool)
		}
		g.GeneratedLinuxImages[image]["[k3s-release(rancher)]"] = true
	}

	r.Source = kdmimages.RKE2
	r.Data = data.RKE2
	rke2ReleaseImages, err := r.GetImages()
	if err != nil {
		return fmt.Errorf("generateFromKDMData: %w", err)
	}
	for _, image := range rke2ReleaseImages {
		if g.GeneratedLinuxImages[image] == nil {
			g.GeneratedLinuxImages[image] = make(map[string]bool)
		}
		g.GeneratedLinuxImages[image]["[rke2-release(rancher)]"] = true
	}

	// get system-images
	s := kdmimages.SystemImages{
		RancherVersion:    g.RancherVersion,
		RkeSysImages:      data.K8sVersionRKESystemImages,
		LinuxSvcOptions:   data.K8sVersionServiceOptions,
		WindowsSvcOptions: data.K8sVersionWindowsServiceOptions,
		RancherVersions:   data.K8sVersionInfo,
	}
	err = s.GetImages()
	if err != nil {
		return fmt.Errorf("generateFromKDMData: %w", err)
	}
	// clone generated system-images
	for image := range s.LinuxImageSet {
		for source := range s.LinuxImageSet[image] {
			u.AddSourceToImage(g.GeneratedLinuxImages, image, source)
		}
	}
	for image := range s.WindowsImageSet {
		for source := range s.WindowsImageSet[image] {
			u.AddSourceToImage(g.GeneratedLinuxImages, image, source)
		}
	}

	// get k3s/rke2 upgrade images
	u := kdmimages.UpgradeImages{
		Source:         kdmimages.K3S,
		RancherVersion: g.RancherVersion,
		MinKubeVersion: g.MinKubeVersion,
		Data:           data.K3S,
	}
	k3sUpgradeImages, err := u.GetImages()
	if err != nil {
		return fmt.Errorf("generateFromKDMData: %w", err)
	}

	for _, image := range k3sUpgradeImages {
		if g.GeneratedLinuxImages[image] == nil {
			g.GeneratedLinuxImages[image] = make(map[string]bool)
		}
		g.GeneratedLinuxImages[image]["k3sUpgrade"] = true
	}

	// 2.5.X does not have RKE2 system images to generate, skip
	if !utils.SemverMajorMinorEqual(g.RancherVersion, "v2.5") {
		u.Source = kdmimages.RKE2
		u.Data = data.RKE2
		rke2UpgradeImages, err := u.GetImages()
		if err != nil {
			return fmt.Errorf("generateFromKDMData: %w", err)
		}
		for _, image := range rke2UpgradeImages {
			if g.GeneratedLinuxImages[image] == nil {
				g.GeneratedLinuxImages[image] = make(map[string]bool)
			}
			g.GeneratedLinuxImages[image]["rke2All"] = true
		}
	}

	return nil
}

func (g *Generator) handleImageArguments() error {
	return nil
}
