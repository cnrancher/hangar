package scan

import (
	"context"
	"errors"

	"github.com/aquasecurity/trivy/pkg/log"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/opencontainers/go-digest"
)

const (
	DefaultDBRepository     = "ghcr.io/aquasecurity/trivy-db"
	DefaultJavaDBRepository = "ghcr.io/aquasecurity/trivy-java-db"
)

type Scanner interface {
	Scan(context.Context, *Option) (*ImageResult, error)
}

// Option is the option when scanning container image
type Option struct {
	ReferenceName string
	Digest        digest.Digest
	Platform      Platform
}

// ScannerOption is the option for creating the global image scanner
type ScannerOption struct {
	TrivyServerURL        string
	Offline               bool
	InsecureSkipTLSVerify bool
	CacheDirectory        string

	// Output format: json, yaml, csv, spdx-json
	Format string
	// Scanners: vuln, misconfig, secret, rbac, license, none
	Scanners []string
}

func NewScanner(o ScannerOption) (Scanner, error) {
	if o.CacheDirectory == "" {
		o.CacheDirectory = utils.TrivyCacheDir()
	}

	if o.TrivyServerURL != "" {
		return newRemoteScanner(&o)
	}
	return newImageScanner(&o)
}

func InitTrivyLogOutput(debug, disable bool) {
	log.InitLogger(debug, disable)
}

var (
	globalScanner            Scanner
	ErrScannerNotInitialized = errors.New("scanner not initialized")
)

func InitScanner(o ScannerOption) error {
	if globalScanner != nil {
		return nil
	}

	var err error
	globalScanner, err = NewScanner(o)
	if err != nil {
		return err
	}
	return nil
}

func Scan(ctx context.Context, o *Option) (*ImageResult, error) {
	if !dbInitialized {
		return nil, ErrDBNotInitialized
	}
	if globalScanner == nil {
		return nil, ErrScannerNotInitialized
	}

	report, err := globalScanner.Scan(ctx, o)
	if err != nil {
		return nil, err
	}
	return report, nil
}
