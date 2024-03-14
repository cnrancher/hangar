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
	"github.com/sirupsen/logrus"
)

type remoteScanner struct {
	trivyServerURL        string
	insecureSkipTLSVerify bool
	offline               bool
	cacheDirectory        string

	cache         fcache.Cache
	clientScanner *client.Scanner
}

func newRemoteScanner(o *Option) (*remoteScanner, error) {
	logrus.Debugf("Create scanner with options %v", utils.PrintObject(o))
	s := &remoteScanner{
		trivyServerURL:        o.TrivyServerURL,
		insecureSkipTLSVerify: o.InsecureSkipTLSVerify,
		offline:               o.Offline,
		cacheDirectory:        o.CacheDirectory,
	}
	s.initCache()
	s.initClientScanner()

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

func (s *remoteScanner) Scan(
	ctx context.Context, refName string,
) (*ImageResult, error) {
	logrus.Debugf("Start to scan image %q", refName)
	if !dbInitialized {
		return nil, ErrDBNotInitialized
	}
	typesImage, cleanup, err := image.NewContainerImage(ctx, refName, ftypes.ImageOptions{
		RegistryOptions: ftypes.RegistryOptions{
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
		analyzer.TypeExecutable, // Disable SBOM
	}
	disabledAnalyzers = append(disabledAnalyzers, analyzer.TypeConfigFiles...)
	artifactArtifact, err := artifactimage.NewArtifact(typesImage, s.cache, artifact.Option{
		DisabledAnalyzers: disabledAnalyzers,
		DisabledHandlers:  nil,
		SkipFiles:         nil,
		SkipDirs:          nil,
		FilePatterns:      nil,
		NoProgress:        false,
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
	})
	if err != nil {
		return nil, fmt.Errorf("artifactimage.NewArtifact failed: %v", err)
	}

	ss := scanner.NewScanner(s.clientScanner, artifactArtifact)
	logrus.Debugf("Start scan artifact")
	report, err := ss.ScanArtifact(ctx, types.ScanOptions{
		VulnType: types.VulnTypes,
		Scanners: types.Scanners{
			types.VulnerabilityScanner, // TODO: configurable
			// types.SecretScanner,
		},
		ImageConfigScanners: nil,
		ScanRemovedPackages: false,
		ListAllPackages:     false,
		// LicenseCategories:   types.AllImageConfigScanners,
		FilePatterns:   nil,
		IncludeDevDeps: false,
	})
	if err != nil {
		return nil, fmt.Errorf("scanArtifact failed: %w", err)
	}

	imageResult := NewImageResult(&report, "", Platform{})
	return imageResult, nil
}
