package oci

import (
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
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"

	signaturev5 "github.com/containers/image/v5/signature"
	typesv5 "github.com/containers/image/v5/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Chart struct {
	url     string
	name    string
	version string

	insecureSkipVerify bool
	systemContext      *typesv5.SystemContext
	policy             *signaturev5.Policy

	cacheDir    string
	imageDir    string
	imageSource string
}

type ChartOptions struct {
	URL     string
	Name    string
	Version string

	InsecureSkipVerify bool
	SystemContext      *typesv5.SystemContext
	Policy             *signaturev5.Policy
}

func NewChart(opts *ChartOptions) *Chart {
	return &Chart{
		url:                opts.URL,
		name:               opts.Name,
		version:            opts.Version,
		insecureSkipVerify: opts.InsecureSkipVerify,
		systemContext:      utils.CopySystemContext(opts.SystemContext),
		policy:             opts.Policy,
	}
}

// Fetch chart from remote URL or local directory and convert it to OCI image.
// Need to use [Chart.Cleanup] method manually to delete the cache directory.
func (c *Chart) Fetch(ctx context.Context) error {
	mode, err := os.Stat(c.url)
	switch {
	case err == nil && mode.IsDir():
		return c.fromDirectory()
	case registry.IsOCI(c.url):
		return c.fromOCI(ctx)
	case strings.HasPrefix(c.url, "http://") || strings.HasPrefix(c.url, "https://"):
		if strings.HasSuffix(c.url, ".tgz") || strings.HasSuffix(c.url, ".tar.gz") {
			return c.fromURL(ctx)
		}
		return c.fromRepo(ctx)
	default:
		return fmt.Errorf("invalid URL format %q", c.url)
	}
}

func (c *Chart) CacheDir() string {
	return c.cacheDir
}

func (c *Chart) ImageDir() string {
	return c.imageDir
}

func (c *Chart) image() (*archive.Image, error) {
	inspector, err := manifest.NewInspector(context.TODO(), &manifest.InspectorOption{
		ReferenceName: fmt.Sprintf("oci:%v", c.imageDir),
		SystemContext: utils.SystemContextWithSharedBlobDir(
			c.systemContext, filepath.Join(c.cacheDir, archive.SharedBlobDir)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create inspector from dir %q: %w",
			c.imageDir, err)
	}
	defer inspector.Close()
	// Use background context to inspect local directory
	b, mime, err := inspector.Raw(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to inspect dir %q: %w",
			c.imageDir, err)
	}
	manifestDigest := digest.SHA256.FromBytes(b)
	if mime != imgspecv1.MediaTypeImageManifest {
		return nil, fmt.Errorf("image MIME type should be OCI image manifest v1, got %q", mime)
	}
	manifest := &imgspecv1.Manifest{}
	if err := json.Unmarshal(b, manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}
	spec := archive.ImageSpec{
		Arch:        "",
		OS:          "",
		OSVersion:   "",
		OSFeatures:  nil,
		Variant:     "",
		MediaType:   mime,
		Layers:      nil,
		Config:      manifest.Config.Digest,
		Digest:      manifestDigest,
		Annotations: manifest.Annotations,
	}
	for _, layer := range manifest.Layers {
		spec.Layers = append(spec.Layers, layer.Digest)
	}
	image := &archive.Image{
		Source:   c.Source(),
		Tag:      c.version,
		ArchList: make([]string, 0),
		OsList:   make([]string, 0),
		Images:   []archive.ImageSpec{spec},
	}
	return image, nil
}

func (c *Chart) Source() string {
	return c.imageSource
}

// WriteArchive writes chart OCI image into the archive file.
func (c *Chart) WriteArchive(au *archive.Updater) error {
	if c.cacheDir == "" {
		return fmt.Errorf("chart is not fetched to the cache directory")
	}
	err := au.Append(c.cacheDir)
	if err != nil {
		return fmt.Errorf("write dir %q to archive file: %w",
			c.cacheDir, err)
	}
	index := au.Index()
	image, err := c.image()
	if err != nil {
		return err
	}
	index.Append(image)
	au.SetIndex(index)
	return au.UpdateIndex()
}

func (c *Chart) Cleanup() error {
	if c.cacheDir == "" {
		return nil
	}
	return os.RemoveAll(c.cacheDir)
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
	c.imageSource = image
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
	c.imageSource = chartURL
	c.cacheDir, c.imageDir, err = fromURL(ctx, chartURL, c.insecureSkipVerify)
	return err
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
	c.imageSource = chartPath
	c.cacheDir, c.imageDir, err = fromReader(f)
	return err
}

func (c *Chart) fromURL(ctx context.Context) error {
	var err error
	c.imageSource = c.url
	c.cacheDir, c.imageDir, err = fromURL(ctx, c.url, c.insecureSkipVerify)
	return err
}

func fromURL(
	ctx context.Context, url string, insecureSkipVerify bool,
) (cache string, image string, err error) {
	logrus.Debugf("Get helm chart from %q", url)
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := utils.HTTPClientDoWithRetry(ctx, client, req)
	if err != nil {
		return "", "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("failed to get url %q: %v", url, resp.Status)
	}
	cache, image, err = fromReader(resp.Body)
	return
}

func fromReader(r io.Reader) (cache string, image string, err error) {
	layerData, err := io.ReadAll(r)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch chart data: %w", err)
	}
	cd, err := newFileCacheDir()
	if err != nil {
		return "", "", err
	}

	layerDigest := digest.SHA256.FromBytes(layerData)
	tmpChartName := fmt.Sprintf("sha256-%x.tgz", layerDigest.Encoded())
	tmpChartPath := filepath.Join(cd, tmpChartName)
	f, err := os.Create(tmpChartPath)
	if err != nil {
		return cd, "", fmt.Errorf("failed to create %q: %w", tmpChartPath, err)
	}
	defer f.Close()
	_, err = f.Write(layerData)
	if err != nil {
		return cd, "", fmt.Errorf("failed to write %q: %w", tmpChartPath, err)
	}

	c, err := loader.Load(tmpChartPath)
	if err != nil {
		return cd, "", fmt.Errorf("failed to load chart: %w", err)
	}
	configData, err := json.MarshalIndent(c.Metadata, "", "  ")
	if err != nil {
		return cd, "", fmt.Errorf("failed to marshal chart metadata: %w", err)
	}
	configDigest := digest.SHA256.FromBytes(configData)

	m := &imgspecv1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: imgspecv1.MediaTypeImageManifest,
		Config: imgspecv1.Descriptor{
			MediaType: registry.ConfigMediaType,
			Size:      int64(len(configData)),
			Digest:    configDigest,
		},
		Layers: []imgspecv1.Descriptor{
			{
				MediaType: registry.ChartLayerMediaType,
				Size:      int64(len(layerData)),
				Digest:    layerDigest,
			},
		},
		Annotations: map[string]string{
			imgspecv1.AnnotationURL:         c.Metadata.Home,
			imgspecv1.AnnotationVersion:     c.Metadata.Version,
			imgspecv1.AnnotationTitle:       c.Metadata.Name,
			imgspecv1.AnnotationDescription: c.Metadata.Description,
		},
	}
	manifestData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return cd, "", fmt.Errorf("failed to marshal manifest: %w", err)
	}
	manifestDigest := digest.SHA256.FromBytes(manifestData)

	// Construct chart file to containers OCI format
	err = os.Mkdir(filepath.Join(cd, manifestDigest.Encoded()), 0755)
	if err != nil {
		return cd, "", fmt.Errorf("mkdir failed on %q: %w",
			filepath.Join(cd, manifestDigest.Encoded()), err)
	}
	// Create share blobs path
	sharedBlobPath := filepath.Join(cd, archive.SharedBlobDir, "sha256")
	if err = os.MkdirAll(sharedBlobPath, 0755); err != nil {
		return cd, "", fmt.Errorf("mkdir failed %q: %w", sharedBlobPath, err)
	}
	// Chart layer
	if err = os.Rename(tmpChartPath,
		filepath.Join(sharedBlobPath, layerDigest.Encoded())); err != nil {
		return cd, "", fmt.Errorf("chart layer file rename failed: %w", err)
	}
	// Manifest file
	if err = os.WriteFile(filepath.Join(sharedBlobPath, manifestDigest.Encoded()), manifestData, 0644); err != nil {
		return cd, "", fmt.Errorf("failed to write manifest: %w", err)
	}
	// Config file
	if err = os.WriteFile(filepath.Join(sharedBlobPath, configDigest.Encoded()), configData, 0644); err != nil {
		return cd, "", fmt.Errorf("failed to write config: %w", err)
	}
	// OCI Image index
	imagePath := filepath.Join(cd, manifestDigest.Encoded())
	if err = os.MkdirAll(filepath.Join(imagePath, imgspecv1.ImageBlobsDir), 0755); err != nil {
		return cd, "", fmt.Errorf("mkdir failed %q: %w", imagePath, err)
	}
	index := imgspecv1.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Manifests: []imgspecv1.Descriptor{
			{
				MediaType: imgspecv1.MediaTypeImageManifest,
				Digest:    manifestDigest,
				Size:      int64(len(manifestData)),
			},
		},
	}
	indexData, err := json.Marshal(index)
	if err != nil {
		return cd, imagePath, fmt.Errorf("failed to marshal manifest index: %w", err)
	}
	if err := os.WriteFile(
		filepath.Join(imagePath, imgspecv1.ImageIndexFile), indexData, 0644); err != nil {
		return cd, imagePath, fmt.Errorf("failed to write manifest index: %w", err)
	}
	// Image Layout
	layout := imgspecv1.ImageLayout{
		Version: imgspecv1.ImageLayoutVersion,
	}
	layoutData, err := json.Marshal(layout)
	if err != nil {
		return cd, imagePath, fmt.Errorf("failed to marshal image layout: %w", err)
	}
	if err := os.WriteFile(
		filepath.Join(imagePath, imgspecv1.ImageLayoutFile), layoutData, 0644); err != nil {
		return cd, imagePath, fmt.Errorf("failed to write layout file: %w", err)
	}
	return cd, imagePath, nil
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
