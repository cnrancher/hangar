package scan

import (
	"context"
	"errors"

	"github.com/aquasecurity/trivy/pkg/log"
	"github.com/cnrancher/hangar/pkg/utils"
)

const (
	DefaultDBRepository     = "ghcr.io/aquasecurity/trivy-db"
	DefaultJavaDBRepository = "ghcr.io/aquasecurity/trivy-java-db"
)

type Scanner interface {
	Scan(context.Context, string) (*ImageResult, error)
}

type ScanOptions struct {
	ReferenceName string
}

type Option struct {
	TrivyServerURL        string
	Offline               bool
	InsecureSkipTLSVerify bool
	CacheDirectory        string
}

func NewScanner(o Option) (Scanner, error) {
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

func InitScanner(o Option) error {
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

func Scan(ctx context.Context, refName string) (*ImageResult, error) {
	if !dbInitialized {
		return nil, ErrDBNotInitialized
	}
	if globalScanner == nil {
		return nil, ErrScannerNotInitialized
	}

	report, err := globalScanner.Scan(ctx, refName)
	if err != nil {
		return nil, err
	}
	return report, nil
}
