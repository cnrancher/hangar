package scan

import (
	"context"
	"errors"
	"fmt"

	"github.com/aquasecurity/trivy/pkg/db"
	"github.com/aquasecurity/trivy/pkg/javadb"
	"github.com/aquasecurity/trivy/pkg/log"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/opencontainers/go-digest"
)

var (
	DefaultECRRepository = fmt.Sprintf("%s:%d",
		"public.ecr.aws/aquasecurity/trivy-db", db.SchemaVersion)
	DefaultJavaECRRepository = fmt.Sprintf("%s:%d",
		"public.ecr.aws/aquasecurity/trivy-java-db", javadb.SchemaVersion)

	DefaultGHCRRepository     = db.DefaultGHCRRepository
	DefaultJavaGHCRRepository = javadb.DefaultGHCRRepository
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

func NewScanner(ctx context.Context, o ScannerOption) (Scanner, error) {
	if o.CacheDirectory == "" {
		o.CacheDirectory = utils.TrivyCacheDir()
	}

	if o.TrivyServerURL != "" {
		return newRemoteScanner(ctx, &o)
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

func InitScanner(ctx context.Context, o ScannerOption) error {
	if globalScanner != nil {
		return nil
	}

	var err error
	globalScanner, err = NewScanner(ctx, o)
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
