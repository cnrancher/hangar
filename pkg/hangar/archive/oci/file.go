package oci

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/utils"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"helm.sh/helm/v3/pkg/registry"
)

const (
	DefaultFileProject = "hangar-file"

	FileLayerMediaType = "application/vnd.content.hangar.file.layer.v1"
)

type File struct {
	common

	url string
}

type FileOptions struct {
	CommonOpts

	URL string
}

type FileConfig struct {
	Name string
}

func NewFile(opts *FileOptions) *File {
	return &File{
		common: common{
			insecureSkipVerify: opts.CommonOpts.InsecureSkipVerify,
			systemContext:      utils.CopySystemContext(opts.CommonOpts.SystemContext),
			policy:             opts.CommonOpts.Policy,
		},
		url: opts.URL,
	}
}

// Fetch file from remote URL or local directory and convert it to OCI image.
// Need to use [common.Cleanup] method manually to delete the cache directory.
func (f *File) Fetch(ctx context.Context) error {
	name := filepath.Base(f.url)
	f.imageSource = fmt.Sprintf("%v/%v/%v",
		utils.DockerHubRegistry, DefaultFileProject, name)
	f.imageTag = utils.DefaultTag

	mode, err := os.Stat(f.url)
	switch {
	case err == nil:
		if mode.IsDir() {
			return fmt.Errorf("%v is a directory", f.url)
		}
		return f.fromDirectory()
	case registry.IsOCI(f.url):
		return fmt.Errorf("file protocol does not support Helm OCI image")
	case strings.HasPrefix(f.url, "http://") || strings.HasPrefix(f.url, "https://"):
		return f.fromURL(ctx)
	default:
		return fmt.Errorf("invalid URL %q: %w", f.url, err)
	}
}

func (f *File) fromDirectory() error {
	file, err := os.Open(f.url)
	if err != nil {
		return fmt.Errorf("failed to open file %q", f.url)
	}
	defer file.Close()
	return f.fromReader(file)
}

func (f *File) fromURL(ctx context.Context) error {
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: f.insecureSkipVerify},
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := utils.HTTPClientDoWithRetry(ctx, client, req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to get url %q: %v", f.url, resp.Status)
	}
	return f.fromReader(resp.Body)
}

// fromReader constructs the Hangar OCI format File image into
// the cache directory from [io.Reader].
func (f *File) fromReader(r io.Reader) (err error) {
	fileName := filepath.Base(f.url)
	f.annotations = map[string]string{
		imgspecv1.AnnotationTitle:       fileName,
		imgspecv1.AnnotationDescription: f.url,
		imgspecv1.AnnotationVendor:      "Hangar",
		imgspecv1.AnnotationCreated:     time.Now().Format(time.RFC3339),
	}
	f.config = &FileConfig{
		Name: fileName,
	}
	f.layerMediaType = FileLayerMediaType

	if err := f.constructOCIImage(r); err != nil {
		return fmt.Errorf("failed to construct OCI image from file %q: %w",
			f.url, err)
	}
	return nil
}
