package oci

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/image/destination"
	"github.com/cnrancher/hangar/pkg/image/manifest"
	"github.com/cnrancher/hangar/pkg/image/source"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/rancher/chartimages"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	DefaultChartProject = "hangar-helm"
)

type Chart struct {
	common

	url     string
	name    string
	version string
}

type ChartOptions struct {
	CommonOpts
	URL     string
	Name    string
	Version string
}

func NewChart(opts *ChartOptions) *Chart {
	return &Chart{
		common: common{
			insecureSkipVerify: opts.CommonOpts.InsecureSkipVerify,
			systemContext:      utils.CopySystemContext(opts.CommonOpts.SystemContext),
			policy:             opts.CommonOpts.Policy,
		},
		url:     opts.URL,
		name:    opts.Name,
		version: opts.Version,
	}
}

// Fetch chart from remote URL or local directory and convert it to OCI image.
// Need to use [common.Cleanup] method manually to delete the cache directory.
func (c *Chart) Fetch(ctx context.Context) error {
	mode, err := os.Stat(c.url)
	switch {
	case err == nil:
		if mode.IsDir() {
			return c.fromDirectory()
		}
		return c.fromFile()
	case registry.IsOCI(c.url):
		return c.fromOCI(ctx)
	case strings.HasPrefix(c.url, "http://") || strings.HasPrefix(c.url, "https://"):
		if strings.HasSuffix(c.url, ".tgz") || strings.HasSuffix(c.url, ".tar.gz") {
			return c.fromURL(ctx, c.url)
		}
		return c.fromRepo(ctx)
	default:
		return fmt.Errorf("invalid chart %q: %w", c.url, err)
	}
}

func (c *Chart) fromOCI(ctx context.Context) error {
	image, err := url.JoinPath(c.url, c.name)
	if err != nil {
		return fmt.Errorf("failed to construct image: %w", err)
	}
	image = strings.TrimPrefix(image, fmt.Sprintf("%v://", registry.OCIScheme))
	if c.version != "" {
		image = fmt.Sprintf("%v:%v", image, c.version)
	}
	logrus.Debugf("OCI helm chart image: %v", image)
	src, err := source.NewSource(&source.Option{
		Type:          types.TypeDocker,
		Registry:      utils.GetRegistryName(image),
		Project:       utils.GetProjectName(image),
		Name:          utils.GetImageName(image),
		Tag:           utils.GetImageTag(image),
		SystemContext: utils.SystemContextWithTLSVerify(c.systemContext, c.insecureSkipVerify),
	})
	if err != nil {
		return fmt.Errorf("failed to create source image: %w", err)
	}
	if err := src.Init(ctx); err != nil {
		return fmt.Errorf("failed to init source image: %w", err)
	}

	cd, err := newFileCacheDir()
	if err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}
	c.cacheDir = cd
	c.imageDir = filepath.Join(c.cacheDir, src.ManifestDigest().Encoded())
	sd := filepath.Join(cd, archive.SharedBlobDir)
	dest, err := destination.NewDestination(&destination.Option{
		Type:      types.TypeOci,
		Directory: c.cacheDir,
		Name:      utils.GetImageName(image),
		Tag:       utils.GetImageTag(image),
		SystemContext: utils.SystemContextWithSharedBlobDir(
			c.systemContext, sd),
	})
	if err != nil {
		os.RemoveAll(cd)
		return fmt.Errorf("failed to create dest image: %w", err)
	}
	if err := dest.Init(ctx); err != nil {
		return fmt.Errorf("failed to init dest image: %w", err)
	}

	err = src.Copy(ctx, &source.CopyOptions{
		CopyProvenance:     false,
		SigstorePrivateKey: "",
		SigstorePassphrase: nil,
		Destination:        dest,
		Set:                make(types.FilterSet),
		Policy:             c.policy,
	})
	if err != nil {
		return fmt.Errorf("failed to copy image %q to %q: %w",
			src.ReferenceName(), dest.ReferenceName(), err)
	}

	inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
		ReferenceName: fmt.Sprintf("oci:%v", filepath.Join(c.cacheDir, src.ManifestDigest().Encoded())),
		SystemContext: dest.SystemContext(),
	})
	if err != nil {
		return fmt.Errorf("failed to create dest image inspector: %w", err)
	}
	b, _, err := inspector.Raw(ctx)
	if err != nil {
		return fmt.Errorf("failed to inspect dest image: %w", err)
	}
	destManifest := &imgspecv1.Manifest{}
	if err := json.Unmarshal(b, destManifest); err != nil {
		return fmt.Errorf("failed to unmarshal dest image manifest: %w", err)
	}
	c.manifestDigest = digest.SHA256.FromBytes(b)
	c.configDigest = destManifest.Config.Digest
	c.annotations = destManifest.Annotations
	for _, l := range destManifest.Layers {
		c.layers = append(c.layers, l.Digest)
	}
	c.imageTag = c.version
	if c.imageTag == "" {
		c.imageTag = utils.DefaultTag
	}
	c.imageSource = fmt.Sprintf("%v/%v/%v",
		utils.GetRegistryName(image),
		utils.GetProjectName(image),
		utils.GetImageName(image))
	return nil
}

func (c *Chart) fromRepo(ctx context.Context) error {
	logrus.Debugf("Get helm chart %q version %q from %q",
		c.name, c.version, c.url)
	if c.name == "" {
		return fmt.Errorf("chart name not provided")
	}
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: c.insecureSkipVerify},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	indexURL, err := url.JoinPath(c.url, "index.yaml")
	if err != nil {
		return fmt.Errorf("failed to join index URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := utils.HTTPClientDoWithRetry(ctx, client, req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to request index %q: %v", indexURL, resp.Status)
	}
	indexData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to fetch index data from %q: %w", c.url, err)
	}
	indexCacheDir, err := newFileCacheDir()
	if err != nil {
		return fmt.Errorf("failed to init index cache dir: %w", err)
	}
	defer os.RemoveAll(indexCacheDir)
	if err := os.WriteFile(filepath.Join(indexCacheDir, "index.yaml"), indexData, 0600); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}
	index, err := repo.LoadIndexFile(filepath.Join(indexCacheDir, "index.yaml"))
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}
	chartVersion, err := findChartVersion(index, c.name, c.version)
	if err != nil {
		return fmt.Errorf("failed to get chart %q, version %q: %w",
			c.name, c.version, err)
	}
	chartURL, err := url.JoinPath(c.url, chartVersion.URLs[0])
	if err != nil {
		return fmt.Errorf("failed to get chart URL: %w", err)
	}
	return c.fromURL(ctx, chartURL)
}

func (c *Chart) fromFile() error {
	f, err := os.Open(c.url)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	return c.fromReader(f)
}

func (c *Chart) fromDirectory() error {
	repoIndex, err := chartimages.BuildOrGetIndex(c.url)
	if err != nil {
		return fmt.Errorf("failed to get repo index: %w", err)
	}

	chartVersion, err := findChartVersion(repoIndex, c.name, c.version)
	if err != nil {
		return fmt.Errorf("failed to get chart %q, version %q: %w",
			c.name, c.version, err)
	}

	chartPath := filepath.Join(c.url, chartVersion.URLs[0])
	f, err := os.Open(chartPath)
	if err != nil {
		return fmt.Errorf("failed to open chart file %q", chartPath)
	}
	defer f.Close()
	return c.fromReader(f)
}

func (c *Chart) fromURL(ctx context.Context, url string) error {
	logrus.Debugf("Get helm chart from %q", c.url)
	var err error

	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: c.insecureSkipVerify},
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := utils.HTTPClientDoWithRetry(ctx, client, req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to get url %q: %v", c.url, resp.Status)
	}
	return c.fromReader(resp.Body)
}

// fromReader constructs the Hangar OCI format Chart image into
// the cache directory from [io.Reader].
func (c *Chart) fromReader(r io.Reader) (err error) {
	var buffer bytes.Buffer
	teeReader := io.TeeReader(r, &buffer)
	chart, err := loadChartFromReader(teeReader)
	if err != nil {
		return fmt.Errorf("load chart: %w", err)
	}
	metadata := chart.Metadata
	c.annotations = map[string]string{
		imgspecv1.AnnotationVendor:      utils.Name,
		imgspecv1.AnnotationCreated:     time.Now().Format(time.RFC3339),
		imgspecv1.AnnotationURL:         metadata.Home,
		imgspecv1.AnnotationVersion:     metadata.Version,
		imgspecv1.AnnotationTitle:       metadata.Name,
		imgspecv1.AnnotationDescription: metadata.Description,
	}
	c.config = metadata
	c.layerMediaType = registry.ChartLayerMediaType
	c.name = metadata.Name
	c.imageSource = fmt.Sprintf("%v/%v/%v",
		utils.DockerHubRegistry, DefaultChartProject, metadata.Name)
	c.imageTag = metadata.Version
	if c.imageTag == "" {
		c.imageTag = utils.DefaultTag
	}
	c.imageTag = strings.ReplaceAll(c.imageTag, "+", "-")
	if err := c.constructOCIImage(&buffer); err != nil {
		return fmt.Errorf("failed to construct OCI image from file %q: %w",
			c.url, err)
	}
	return nil
}

func findChartVersion(
	index *repo.IndexFile, name string, version string,
) (*repo.ChartVersion, error) {
	if name == "" {
		return nil, fmt.Errorf("chart name not provided")
	}
	var versions repo.ChartVersions
	for n, v := range index.Entries {
		if n == name {
			versions = v
			break
		}
	}
	if versions == nil {
		return nil, fmt.Errorf("chart %q not found in repository", name)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("chart %q does not have any versions in repo", name)
	}
	var chartVersion *repo.ChartVersion
	if version != "" {
		expectedVersion, err := semver.NewVersion(version)
		if err != nil {
			return nil, fmt.Errorf("failed to parse version %q: %w", version, err)
		}
		for _, v := range versions {
			v1, err := semver.NewVersion(v.Version)
			if err != nil {
				continue
			}
			if v1.Equal(expectedVersion) {
				chartVersion = v
				break
			}
		}
		if chartVersion == nil {
			return nil, fmt.Errorf("failed to find chart %q version %q", name, version)
		}
	} else {
		chartVersion = versions[0]
	}

	if len(chartVersion.URLs) == 0 {
		return nil, fmt.Errorf("chart %q version %q does not have URLs provided",
			name, version)
	}
	return chartVersion, nil
}

func loadChartFromReader(r io.Reader) (*chart.Chart, error) {
	cd, err := newFileCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to create tmp dir: %w", err)
	}
	defer os.RemoveAll(cd)

	f, err := os.CreateTemp(cd, "tmp-*.tgz")
	if err != nil {
		return nil, fmt.Errorf("failed to create tmp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return nil, fmt.Errorf("failed to write tmp file %q: %w", f.Name(), err)
	}
	tmpFilePath := f.Name() // name is the absolute path of the file
	chart, err := loader.Load(tmpFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart %q: %w", tmpFilePath, err)
	}
	return chart, nil
}
