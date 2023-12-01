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

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/cnrancher/hangar/pkg/rancher/chartimages"
	"github.com/cnrancher/hangar/pkg/rancher/kdmimages"
	u "github.com/cnrancher/hangar/pkg/utils"
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

func (g *Generator) Generate(ctx context.Context) error {
	if err := g.selfCheck(); err != nil {
		return err
	}
	g.init()

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

	if err := g.handleImageArguments(ctx); err != nil {
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
				u.AddSourceToImage(g.GeneratedLinuxImages, image, source)
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
				u.AddSourceToImage(g.GeneratedWindowsImages, image, source)
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
			for source := range c.ImageSet[image] {
				u.AddSourceToImage(g.GeneratedLinuxImages, image, source)
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
				u.AddSourceToImage(g.GeneratedWindowsImages, image, source)
			}
		}
		// Delete cloned chart path after generated images
		logrus.Debugf("Delete %q", u.CacheCloneRepoDirectory)
		if err := u.DeleteIfExist(u.CacheCloneRepoDirectory); err != nil {
			return err
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
	logrus.Infof("get KDM data from URL: %q", g.KDMURL)
	b, err := getHTTPData(ctx, g.KDMURL, time.Second*30)
	if err != nil {
		// re-try get data from KDM url
		logrus.Warn(err)
		logrus.Warnf("failed to get KDM data, retrying...")
		b, err = getHTTPData(ctx, g.KDMURL, time.Second*30)
		if err != nil {
			return fmt.Errorf("generateFromKDMURL: %w", err)
		}
	}
	return g.generateFromKDMData(ctx, b)
}

func (g *Generator) generateFromKDMData(_ context.Context, b []byte) error {
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
	upgrade := kdmimages.UpgradeImages{
		Source:         kdmimages.K3S,
		RancherVersion: g.RancherVersion,
		MinKubeVersion: g.MinKubeVersion,
		Data:           data.K3S,
	}
	k3sUpgradeImages, err := upgrade.GetImages()
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
	if !u.SemverMajorMinorEqual(g.RancherVersion, "v2.5") {
		upgrade.Source = kdmimages.RKE2
		upgrade.Data = data.RKE2
		rke2UpgradeImages, err := upgrade.GetImages()
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

func (g *Generator) handleImageArguments(_ context.Context) error {
	return nil
}

func getHTTPData(
	ctx context.Context, link string, timeout time.Duration,
) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, fmt.Errorf("getHttpData: %w", err)
	}
	if !cmdconfig.GetBool("tls-verify") {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getHttpData: http.Get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("getHttpData: get url [%q]: %v",
			link, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("getHttpData: io.ReadAll: %w", err)
	}
	return b, nil
}
