package listgenerator

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cnrancher/image-tools/pkg/rancher/chart"
	u "github.com/cnrancher/image-tools/pkg/utils"
	"github.com/rancher/rke/types/kdm"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

// Generator is a generator to generate image list from charts, KDM data, etc.
type Generator struct {
	RancherVersion string // rancher version

	ChartsPaths map[string]chart.ChartRepoType
	ChartURLs   map[string]chart.ChartRepoType

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
		return fmt.Errorf("%q is not a valid version", g.RancherVersion)
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
		c := chart.Chart{
			RancherVersion: g.RancherVersion,
			OS:             chart.Linux,
			Type:           g.ChartsPaths[path],
			Path:           path,
		}
		if err := c.FetchImages(); err != nil {
			logrus.Error(err)
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.GeneratedLinuxImages, image, source)
			}
		}
		// fetch windows images
		c.OS = chart.Windows
		c.ImageSet = make(map[string]map[string]bool)
		if err := c.FetchImages(); err != nil {
			logrus.Error(err)
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
		c := chart.Chart{
			RancherVersion: g.RancherVersion,
			OS:             chart.Linux,
			Type:           g.ChartURLs[url],
			URL:            url,
		}
		if err := c.FetchImages(); err != nil {
			logrus.Error(err)
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.GeneratedLinuxImages, image, source)
			}
		}
		// fetch windows images
		c.OS = chart.Windows
		c.ImageSet = make(map[string]map[string]bool)
		if err := c.FetchImages(); err != nil {
			logrus.Error(err)
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.GeneratedWindowsImages, image, source)
			}
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
		return fmt.Errorf("generateFromKDMPath: %w", err)
	}
	return g.generateFromKDMData(b)
}

func (g *Generator) generateFromKDMURL() error {
	if g.KDMURL == "" {
		return nil
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(g.KDMURL)
	if err != nil {
		return fmt.Errorf("generateFromKDMURL: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("generateFromKDMURL: get url [%q]: %v",
			g.KDMURL, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("generateFromKDMURL: %w", err)
	}
	return g.generateFromKDMData(b)
}

func (g *Generator) generateFromKDMData(b []byte) error {
	data, err := kdm.FromData(b)
	if err != nil {
		return fmt.Errorf("generateFromKDMData: %w", err)
	}
	// get k3s/rke2 upgrade images
	eg := UpgradeGenerator{
		Source:         K3S,
		RancherVersion: g.RancherVersion,
		MinKubeVersion: "v1.21.0",
		Data:           data.K3S,
	}
	k3sUpgradeImages, err := eg.GetImages()
	if err != nil {
		return fmt.Errorf("generateFromKDMData: %w", err)
	}
	sort.Strings(k3sUpgradeImages)

	for _, image := range k3sUpgradeImages {
		if g.GeneratedLinuxImages[image] == nil {
			g.GeneratedLinuxImages[image] = make(map[string]bool)
		}
		g.GeneratedLinuxImages[image]["k3sUpgrade"] = true
	}

	eg.Source = RKE2
	eg.Data = data.RKE2
	rke2UpgradeImages, err := eg.GetImages()
	if err != nil {
		return fmt.Errorf("generateFromKDMData: %w", err)
	}
	sort.Strings(rke2UpgradeImages)
	for _, image := range rke2UpgradeImages {
		if g.GeneratedLinuxImages[image] == nil {
			g.GeneratedLinuxImages[image] = make(map[string]bool)
		}
		g.GeneratedLinuxImages[image]["rke2All"] = true
	}

	return nil
}

func (g *Generator) handleImageArguments() error {
	return nil
}
