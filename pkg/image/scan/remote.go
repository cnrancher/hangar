package scan

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aquasecurity/trivy/pkg/cache"
	"github.com/aquasecurity/trivy/pkg/fanal/analyzer"
	"github.com/aquasecurity/trivy/pkg/fanal/artifact"
	artifactimage "github.com/aquasecurity/trivy/pkg/fanal/artifact/image"
	fcache "github.com/aquasecurity/trivy/pkg/fanal/cache"
	"github.com/aquasecurity/trivy/pkg/fanal/image"
	ftypes "github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/rpc/client"
	"github.com/aquasecurity/trivy/pkg/scanner"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/pkg/docker/config"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

type remoteScanner struct {
	trivyServerURL        string
	insecureSkipTLSVerify bool
	offline               bool
	cacheDirectory        string
	format                string
	scanners              types.Scanners

	cache         fcache.Cache
	clientScanner *client.Scanner
}

func newRemoteScanner(o *ScannerOption) (*remoteScanner, error) {
	logrus.Debugf("Create scanner with options %v", utils.PrintObject(o))
	s := &remoteScanner{
		trivyServerURL:        o.TrivyServerURL,
		insecureSkipTLSVerify: o.InsecureSkipTLSVerify,
		offline:               o.Offline,
		format:                o.Format,
		scanners:              nil,
		cacheDirectory:        o.CacheDirectory,
	}
	s.initCache()
	s.initClientScanner()

	if len(o.Scanners) == 0 {
		// Use default vulnerability scanner if no scanners provided.
		s.scanners = types.Scanners{
			types.VulnerabilityScanner,
		}
		return s, nil
	}
	for _, v := range o.Scanners {
		// Filter invalid scanners
		v := types.Scanner(v)
		switch v {
		// vuln, misconfig, secret, rbac, license, none
		case types.NoneScanner,
			types.VulnerabilityScanner,
			types.MisconfigScanner,
			types.SecretScanner,
			types.RBACScanner,
			types.LicenseScanner:
			s.scanners = append(s.scanners, v)
		}
	}

	return s, nil
}

func (s *remoteScanner) initCache() {
	remoteCache := cache.NewRemoteCache(
		s.trivyServerURL,
		http.Header{},
		s.insecureSkipTLSVerify,
	)
	s.cache = cache.NopCache(remoteCache)
	logrus.Debugf("remote cache of Remote Scanner initialized")
}

func (s *remoteScanner) initClientScanner() {
	clientScanner := client.NewScanner(client.ScannerOption{
		RemoteURL: s.trivyServerURL,
		Insecure:  s.insecureSkipTLSVerify,
	}, []client.Option(nil)...)
	s.clientScanner = &clientScanner
	logrus.Debugf("clientScanner of Remote Scanner initialized")
}

// scanOptions generates the trivy ScanOptions used by scanner.
func (s *remoteScanner) scanOptions() types.ScanOptions {
	so := types.ScanOptions{
		VulnType:            types.VulnTypes,
		Scanners:            s.scanners,
		ImageConfigScanners: nil,
		ScanRemovedPackages: false,
		ListAllPackages:     false,
		LicenseCategories:   nil,
		FilePatterns:        nil,
		IncludeDevDeps:      false,
	}
	switch s.format {
	// Disable scanners and set ListAllPackes to true if the output format is
	// SBOM instead of vulnerabilities.
	case "spdx-json", "spdx-csv":
		so.ListAllPackages = true
		so.Scanners = types.Scanners{
			types.NoneScanner,
		}
		return so
	}
	return so
}

func (s *remoteScanner) Scan(
	ctx context.Context, opt *ScanOption,
) (*ImageResult, error) {
	logrus.Debugf("Start to scan image %q", opt.ReferenceName)
	if !dbInitialized {
		return nil, ErrDBNotInitialized
	}
	registry := utils.GetRegistryName(opt.ReferenceName)
	authConfig, _ := config.GetCredentials(&imagetypes.SystemContext{}, registry)
	typesImage, cleanup, err := image.NewContainerImage(ctx, opt.ReferenceName, ftypes.ImageOptions{
		RegistryOptions: ftypes.RegistryOptions{
			Credentials: []ftypes.Credential{
				{
					Username: authConfig.Username,
					Password: authConfig.Password,
				},
			},
			Insecure: s.insecureSkipTLSVerify,
			Platform: ftypes.Platform{},
		},
		DockerOptions: ftypes.DockerOptions{},
		ImageSources:  ftypes.AllImageSources,
	})
	if err != nil {
		return nil, fmt.Errorf("image.NewContainerImage failed: %w", err)
	}
	defer cleanup()

	disabledAnalyzers := []analyzer.Type{
		analyzer.TypeHistoryDockerfile,
		analyzer.TypeExecutable,
	}
	disabledAnalyzers = append(disabledAnalyzers, analyzer.TypeConfigFiles...)
	ao := artifact.Option{
		DisabledAnalyzers: disabledAnalyzers,
		DisabledHandlers:  nil,
		SkipFiles:         nil,
		SkipDirs:          nil,
		FilePatterns:      nil,
		NoProgress:        true,
		Insecure:          s.insecureSkipTLSVerify,
		Offline:           s.offline,
		SBOMSources:       nil,
		RekorURL:          "https://rekor.sigstore.dev",
		Parallel:          5,
		ImageOption: ftypes.ImageOptions{
			RegistryOptions: ftypes.RegistryOptions{
				Insecure: s.insecureSkipTLSVerify,
			},
			DockerOptions: ftypes.DockerOptions{},
			ImageSources:  ftypes.AllImageSources,
		},
	}
	artifactArtifact, err := artifactimage.NewArtifact(typesImage, s.cache, ao)
	if err != nil {
		return nil, fmt.Errorf("artifactimage.NewArtifact failed: %v", err)
	}

	ss := scanner.NewScanner(s.clientScanner, artifactArtifact)
	logrus.Debugf("Start scan artifact")
	report, err := ss.ScanArtifact(ctx, s.scanOptions())
	if err != nil {
		return nil, fmt.Errorf("scanArtifact failed: %w", err)
	}
	imageResult, err := NewImageResult(ctx, &report, s.format, opt)
	if err != nil {
		return nil, fmt.Errorf("scanArtifact NewImageResult: %w", err)
	}
	return imageResult, nil
}
