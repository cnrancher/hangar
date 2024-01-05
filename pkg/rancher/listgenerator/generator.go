package listgenerator

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/rancher/chartimages"
	"github.com/cnrancher/hangar/pkg/rancher/kdmimages"
	"github.com/cnrancher/hangar/pkg/utils"
	u "github.com/cnrancher/hangar/pkg/utils"
	"github.com/rancher/rke/types/kdm"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

// Generator is a generator to generate image list from charts, KDM data, etc.
type Generator struct {
	RancherVersion string // Rancher version, should be va.b.c
	MinKubeVersion string // Minimum kube verision, should be va.b.c

	ChartsPaths map[string]chartimages.ChartRepoType // map[url]type
	ChartURLs   map[string]struct {
		Type   chartimages.ChartRepoType
		Branch string
	}

	KDMPath string // The path of KDM data.json file.
	KDMURL  string // The remote URL of KDM data.json.

	InsecureSkipVerify bool // Skip TLS Verify.

	// Generated linux images, map[image]map[source]true
	LinuxImages map[string]map[string]bool
	// Generated windows images, map[image]map[source]true
	WindowsImages map[string]map[string]bool

	RKE1Versions map[string]bool // Generated RKE1 versions
	RKE2Versions map[string]bool // Generated RKE2 versions
	K3sVersions  map[string]bool // Generated K3s versions
}

func (g *Generator) init() {
	if g.LinuxImages == nil {
		g.LinuxImages = make(map[string]map[string]bool)
	}
	if g.WindowsImages == nil {
		g.WindowsImages = make(map[string]map[string]bool)
	}
	if g.K3sVersions == nil {
		g.K3sVersions = make(map[string]bool)
	}
	if g.RKE1Versions == nil {
		g.RKE1Versions = make(map[string]bool)
	}
	if g.RKE2Versions == nil {
		g.RKE2Versions = make(map[string]bool)
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

func (g *Generator) Generate(ctx context.Context) error {
	g.init()
	if err := g.selfCheck(); err != nil {
		return err
	}

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
		if err := c.FetchImages(ctx); err != nil {
			return err
		}
		for image := range c.ImageSet {
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.LinuxImages, image, source)
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
				u.AddSourceToImage(g.WindowsImages, image, source)
			}
		}
	}
	return nil
}

func (g *Generator) generateFromChartURLs(ctx context.Context) error {
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
		if err := c.FetchImages(ctx); err != nil {
			return err
		}
		for image := range c.ImageSet {
			if chartimages.IgnoreChartImages[image] {
				continue
			}
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.LinuxImages, image, source)
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
				u.AddSourceToImage(g.WindowsImages, image, source)
			}
		}
	}
	return nil
}

func (g *Generator) generateFromKDMPath(ctx context.Context) error {
	if g.KDMPath == "" {
		return nil
	}
	b, err := os.ReadFile(g.KDMPath)
	if err != nil {
		return err
	}
	return g.generateFromKDMData(ctx, b)
}

func (g *Generator) generateFromKDMURL(ctx context.Context) error {
	if g.KDMURL == "" {
		return nil
	}
	logrus.Infof("Get KDM data from URL: %q", g.KDMURL)

	client := &http.Client{
		Timeout: time.Second * 15,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.InsecureSkipVerify,
			},
			Proxy: http.ProxyFromEnvironment,
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.KDMURL, nil)
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

	getters := []kdmimages.Getter{}
	var getter kdmimages.Getter
	// K3s images getter
	getter, err = kdmimages.NewK3sRKE2Getter(
		kdmimages.K3S, g.RancherVersion, g.MinKubeVersion, data.K3S, g.InsecureSkipVerify)
	if err != nil {
		return err
	}
	getters = append(getters, getter)

	// RKE2 images getter
	getter, err = kdmimages.NewK3sRKE2Getter(
		kdmimages.RKE2, g.RancherVersion, g.MinKubeVersion, data.RKE2, g.InsecureSkipVerify)
	if err != nil {
		return err
	}
	getters = append(getters, getter)

	// RKE images getter
	getter, err = kdmimages.NewRKEGetter(g.RancherVersion, &data)
	if err != nil {
		return err
	}
	getters = append(getters, getter)

	for _, getter := range getters {
		if err = getter.Get(ctx); err != nil {
			return err
		}
		utils.MergeImageSourceSet(g.LinuxImages, getter.LinuxImageSet())
		utils.MergeImageSourceSet(g.WindowsImages, getter.WindowsImageSet())
		// Merge version sets
		switch getter.Source() {
		case kdmimages.RKE:
			utils.MergeSets(g.RKE1Versions, getter.VersionSet())
		case kdmimages.RKE2:
			utils.MergeSets(g.RKE2Versions, getter.VersionSet())
		case kdmimages.K3S:
			utils.MergeSets(g.K3sVersions, getter.VersionSet())
		}
	}
	return nil
}
