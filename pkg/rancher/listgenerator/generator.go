package listgenerator

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/cnrancher/hangar/pkg/rancher/chartimages"
	"github.com/cnrancher/hangar/pkg/rancher/kdmimages"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/rancher/rke/types/kdm"
	"github.com/sirupsen/logrus"
)

type GeneratorOption struct {
	RancherVersion string
	MinKubeVersion string

	ChartsPaths map[string]chartimages.ChartRepoType // map[url]type
	ChartURLs   map[string]struct {
		Type   chartimages.ChartRepoType
		Branch string
	}

	KDMPath string // The path of KDM data.json file.
	KDMURL  string // The remote URL of KDM data.json.

	InsecureSkipTLS     bool
	RemoveDeprecatedKDM bool
}

// Generator is a generator to generate image list from charts, KDM data, etc.
type Generator struct {
	rancherVersion string // Rancher version, should be va.b.c
	minKubeVersion string // Minimum kube verision, should be va.b.c

	chartsPaths map[string]chartimages.ChartRepoType // map[url]type
	chartURLs   map[string]struct {
		Type   chartimages.ChartRepoType
		Branch string
	}

	kdmPath string
	kdmURL  string

	insecureSkipTLS     bool
	removeDeprecatedKDM bool

	// All generated images, map[image]map[source]true
	LinuxImages   map[string]map[string]bool
	WindowsImages map[string]map[string]bool

	RKE1LinuxImages   map[string]map[string]bool
	RKE2LinuxImages   map[string]map[string]bool
	K3sLinuxImages    map[string]map[string]bool
	RKE2WindowsImages map[string]map[string]bool

	RKE1Versions map[string]bool
	RKE2Versions map[string]bool
	K3sVersions  map[string]bool
}

func NewGenerator(o *GeneratorOption) (*Generator, error) {
	if o.RancherVersion == "" {
		return nil, fmt.Errorf("invalid rancher version")
	}
	rancherVersion, err := utils.EnsureSemverValid(o.RancherVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid rancher version: %v", o.RancherVersion)
	}
	if o.ChartURLs == nil && o.ChartsPaths == nil &&
		o.KDMPath == "" && o.KDMURL == "" {
		return nil, fmt.Errorf("no input source provided")
	}

	g := &Generator{
		rancherVersion:      rancherVersion,
		minKubeVersion:      o.MinKubeVersion,
		chartsPaths:         o.ChartsPaths,
		chartURLs:           o.ChartURLs,
		kdmPath:             o.KDMPath,
		kdmURL:              o.KDMURL,
		insecureSkipTLS:     o.InsecureSkipTLS,
		removeDeprecatedKDM: o.RemoveDeprecatedKDM,

		LinuxImages:       make(map[string]map[string]bool),
		WindowsImages:     make(map[string]map[string]bool),
		K3sLinuxImages:    make(map[string]map[string]bool),
		K3sVersions:       make(map[string]bool),
		RKE1LinuxImages:   make(map[string]map[string]bool),
		RKE1Versions:      make(map[string]bool),
		RKE2LinuxImages:   make(map[string]map[string]bool),
		RKE2WindowsImages: make(map[string]map[string]bool),
		RKE2Versions:      make(map[string]bool),
	}
	return g, nil
}

func (g *Generator) Run(ctx context.Context) error {
	if err := g.generateFromChartPaths(ctx); err != nil {
		return err
	}
	if err := g.generateFromChartURLs(ctx); err != nil {
		return err
	}
	if err := g.generateFromKDMPath(ctx); err != nil {
		return err
	}
	if err := g.generateFromKDMURL(ctx); err != nil {
		return err
	}
	return nil
}

func (g *Generator) generateFromChartPaths(ctx context.Context) error {
	if g.chartsPaths == nil || len(g.chartsPaths) == 0 {
		return nil
	}
	for path := range g.chartsPaths {
		c := chartimages.Chart{
			RancherVersion: g.rancherVersion,
			OS:             chartimages.Linux,
			Type:           g.chartsPaths[path],
			Path:           path,
		}
		if err := c.FetchImages(ctx); err != nil {
			return err
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				utils.AddSourceToImage(g.LinuxImages, image, source)
			}
		}
		// fetch windows images
		c.OS = chartimages.Windows
		c.ImageSet = make(map[string]map[string]bool)
		if err := c.FetchImages(ctx); err != nil {
			return err
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				utils.AddSourceToImage(g.WindowsImages, image, source)
			}
		}
	}
	return nil
}

func (g *Generator) generateFromChartURLs(ctx context.Context) error {
	if g.chartURLs == nil || len(g.chartURLs) == 0 {
		return nil
	}
	for url := range g.chartURLs {
		c := chartimages.Chart{
			RancherVersion:  g.rancherVersion,
			OS:              chartimages.Linux,
			Type:            g.chartURLs[url].Type,
			Branch:          g.chartURLs[url].Branch,
			URL:             url,
			InsecureSkipTLS: g.insecureSkipTLS,
		}
		if err := c.FetchImages(ctx); err != nil {
			return err
		}
		for image := range c.ImageSet {
			if chartimages.IgnoreChartImages[image] {
				continue
			}
			for source := range c.ImageSet[image] {
				utils.AddSourceToImage(g.LinuxImages, image, source)
			}
		}
		// fetch windows images
		c.OS = chartimages.Windows
		c.ImageSet = make(map[string]map[string]bool)
		if err := c.FetchImages(ctx); err != nil {
			return err
		}
		for image := range c.ImageSet {
			if chartimages.IgnoreChartImages[image] {
				continue
			}
			for source := range c.ImageSet[image] {
				utils.AddSourceToImage(g.WindowsImages, image, source)
			}
		}
	}
	return nil
}

func (g *Generator) generateFromKDMPath(ctx context.Context) error {
	if g.kdmPath == "" {
		return nil
	}
	b, err := os.ReadFile(g.kdmPath)
	if err != nil {
		return err
	}
	return g.generateFromKDMData(ctx, b)
}

func (g *Generator) generateFromKDMURL(ctx context.Context) error {
	if g.kdmURL == "" {
		return nil
	}
	logrus.Infof("Get KDM data from URL: %q", g.kdmURL)

	client := &http.Client{
		Timeout: time.Second * 15,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.insecureSkipTLS,
			},
			Proxy: http.ProxyFromEnvironment,
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.kdmURL, nil)
	if err != nil {
		return fmt.Errorf("generateFromKDMURL: %w", err)
	}
	resp, err := utils.HTTPClientDoWithRetry(ctx, client, req)
	if err != nil {
		return fmt.Errorf("generateFromKDMURL: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("generateFromKDMURL: %w", err)
	}
	return g.generateFromKDMData(ctx, b)
}

func (g *Generator) generateFromKDMData(ctx context.Context, b []byte) error {
	data, err := kdm.FromData(b)
	if err != nil {
		return fmt.Errorf("generateFromKDMData: %w", err)
	}
	clusters := []kdmimages.ClusterType{
		kdmimages.K3S,
		kdmimages.RKE2,
	}
	if ok, _ := utils.SemverCompare(g.rancherVersion, "v2.12.0-0"); ok < 0 {
		clusters = append(clusters, kdmimages.RKE)
	}
	for _, t := range clusters {
		getter, err := kdmimages.NewGetter(&kdmimages.GetterOptions{
			Type:             t,
			RancherVersion:   g.rancherVersion,
			MinKubeVersion:   g.minKubeVersion,
			KDMData:          data,
			InsecureSkipTLS:  g.insecureSkipTLS,
			RemoveDeprecated: g.removeDeprecatedKDM,
		})
		if err != nil {
			return err
		}

		if err = getter.Get(ctx); err != nil {
			return err
		}
		utils.MergeImageSourceSet(g.LinuxImages, getter.LinuxImageSet())
		utils.MergeImageSourceSet(g.WindowsImages, getter.WindowsImageSet())
		// Merge sets
		switch getter.Source() {
		case kdmimages.RKE:
			utils.MergeSets(g.RKE1Versions, getter.VersionSet())
			utils.MergeImageSourceSet(g.RKE1LinuxImages, getter.LinuxImageSet())
		case kdmimages.RKE2:
			utils.MergeSets(g.RKE2Versions, getter.VersionSet())
			utils.MergeImageSourceSet(g.RKE2LinuxImages, getter.LinuxImageSet())
			// RKE2 supports Windows
			utils.MergeImageSourceSet(g.RKE2WindowsImages, getter.WindowsImageSet())
		case kdmimages.K3S:
			utils.MergeSets(g.K3sVersions, getter.VersionSet())
			utils.MergeImageSourceSet(g.K3sLinuxImages, getter.LinuxImageSet())
		}
	}
	return nil
}
