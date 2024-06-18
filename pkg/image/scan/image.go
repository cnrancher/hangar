package scan

import (
	"context"
	"fmt"

	"github.com/aquasecurity/trivy-db/pkg/db"
	"github.com/aquasecurity/trivy/pkg/fanal/analyzer"
	"github.com/aquasecurity/trivy/pkg/fanal/applier"
	"github.com/aquasecurity/trivy/pkg/fanal/artifact"
	artifactimage "github.com/aquasecurity/trivy/pkg/fanal/artifact/image"
	"github.com/aquasecurity/trivy/pkg/fanal/cache"
	"github.com/aquasecurity/trivy/pkg/fanal/image"
	ftypes "github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/scanner"
	"github.com/aquasecurity/trivy/pkg/scanner/langpkg"
	"github.com/aquasecurity/trivy/pkg/scanner/local"
	"github.com/aquasecurity/trivy/pkg/scanner/ospkg"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/aquasecurity/trivy/pkg/utils/fsutils"
	"github.com/aquasecurity/trivy/pkg/vulnerability"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/pkg/docker/config"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

type imageScanner struct {
	insecureSkipTLSVerify bool
	offline               bool
	cacheDirectory        string
	format                string
	scanners              types.Scanners

	cache        cache.Cache
	localScanner local.Scanner
}

func newImageScanner(o *ScannerOption) (*imageScanner, error) {
	logrus.Debugf("Create scanner with options %v", utils.PrintObject(o))
	s := &imageScanner{
		insecureSkipTLSVerify: o.InsecureSkipTLSVerify,
		offline:               o.Offline,
		cacheDirectory:        o.CacheDirectory,
		format:                o.Format,
		scanners:              nil,
	}
	if err := s.initCache(); err != nil {
		return nil, err
	}
	s.initLocalScanner()

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
		default:
			logrus.Warnf("ignore invalid scanner type %q", v)
		}
	}

	return s, nil
}

func (s *imageScanner) initCache() error {
	fsutils.SetCacheDir(s.cacheDirectory)
	fsCache, err := cache.NewFSCache(s.cacheDirectory)
	if err != nil {
		return fmt.Errorf("unable to initialize fs cache: %w", err)
	}
	s.cache = fsCache
	logrus.Debugf("fs cache of Image Scanner initialized")
	return nil
}

func (s *imageScanner) initLocalScanner() {
	applierApplier := applier.NewApplier(s.cache)
	ospkgScanner := ospkg.NewScanner()
	langpkgScanner := langpkg.NewScanner()
	config := db.Config{}
	client := vulnerability.NewClient(config)
	localScanner := local.NewScanner(applierApplier, ospkgScanner, langpkgScanner, client)
	s.localScanner = localScanner
	logrus.Debugf("localScanner of Image Scanner initialized")
}

// scanOptions generates the trivy ScanOptions used by scanner.
func (s *imageScanner) scanOptions() types.ScanOptions {
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

func (s *imageScanner) Scan(
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
	}
	artifactArtifact, err := artifactimage.NewArtifact(typesImage, s.cache, ao)
	if err != nil {
		return nil, fmt.Errorf("artifactimage.NewArtifact failed: %v", err)
	}

	ss := scanner.NewScanner(s.localScanner, artifactArtifact)
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
