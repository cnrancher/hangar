package scan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aquasecurity/trivy-db/pkg/db"
	trivyjavadb "github.com/aquasecurity/trivy-java-db/pkg/db"
	trivydb "github.com/aquasecurity/trivy/pkg/db"
	ftypes "github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/javadb"
	"github.com/aquasecurity/trivy/pkg/oci"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"

	_ "modernc.org/sqlite" // sqlite driver
)

const (
	mediaType = "application/vnd.aquasec.trivy.javadb.layer.v1.tar+gzip"
)

var (
	dbInitialized       bool
	ErrDBNotInitialized = errors.New("trivy db not initialized")
)

type DBOptions struct {
	TrivyServerURL        string
	CacheDirectory        string
	DBRepository          string
	JavaDBRepository      string
	SkipUpdateDB          bool
	SkipUpdateJavaDB      bool
	InsecureSkipTLSVerify bool
}

func InitTrivyDatabase(ctx context.Context, o DBOptions) error {
	if dbInitialized {
		return nil
	}

	if o.DBRepository == "" {
		o.DBRepository = DefaultDBRepository
	}
	if o.JavaDBRepository == "" {
		o.JavaDBRepository = DefaultJavaDBRepository
	}
	if o.CacheDirectory == "" {
		o.CacheDirectory = utils.TrivyCacheDir()
		os.MkdirAll(o.CacheDirectory, 0700)
	}

	errCh := make(chan error)
	// Init trivy vulnerability database.
	go func() {
		if o.TrivyServerURL != "" {
			errCh <- nil
			return
		}
		err := initDB(ctx, &o)
		errCh <- err
	}()
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	// Init trivy java index database.
	go func() {
		err := initJavaDB(ctx, &o)
		errCh <- err
	}()
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	dbInitialized = true
	return nil
}

// Init trivy java database.
// Mandatory for both trivy client & local scan mode.
func initJavaDB(ctx context.Context, o *DBOptions) error {
	javadb.Init(
		o.CacheDirectory,
		o.JavaDBRepository,
		false, false,
		ftypes.RegistryOptions{
			Insecure: o.InsecureSkipTLSVerify,
		},
	)

	dbDir := filepath.Join(o.CacheDirectory, "java-db")
	repo := fmt.Sprintf("%s:%d", o.JavaDBRepository, trivyjavadb.SchemaVersion)
	metac := trivyjavadb.NewMetadata(dbDir)
	meta, err := metac.Get()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("java DB metadata error: %w", err)
		} else if o.SkipUpdateJavaDB {
			logrus.Error("The first run cannot skip downloading Java DB")
			return fmt.Errorf("java database cannot skip on first run")
		}
	}
	if (meta.Version != trivyjavadb.SchemaVersion || meta.NextUpdate.Before(time.Now().UTC())) && !o.SkipUpdateJavaDB {
		// Download DB
		logrus.Infof("Java DB Repository: %s", repo)
		logrus.Infof("Downloading the Java DB to the cache dir %q", dbDir)

		var a *oci.Artifact
		if a, err = oci.NewArtifact(repo, false, ftypes.RegistryOptions{
			Insecure: o.InsecureSkipTLSVerify,
		}); err != nil {
			return fmt.Errorf("oci.NewArtifact failed: %w", err)
		}
		if err = a.Download(ctx, dbDir, oci.DownloadOption{MediaType: mediaType}); err != nil {
			return fmt.Errorf("failed to download java DB: %w", err)
		}

		// Parse the newly downloaded metadata.json
		meta, err = metac.Get()
		if err != nil {
			return fmt.Errorf("java DB metadata error: %w", err)
		}

		// Update DownloadedAt
		meta.DownloadedAt = time.Now().UTC()
		if err = metac.Update(meta); err != nil {
			return fmt.Errorf("java DB metadata update error: %w", err)
		}
		logrus.Info("The Java DB is cached for 3 days.")
	}
	logrus.Debugf("javaDB initialized")
	return nil
}

func initDB(ctx context.Context, o *DBOptions) error {
	logrus.Debugf("Start creating trivy vulnerability database client.")
	client := trivydb.NewClient(
		o.CacheDirectory, false, trivydb.WithDBRepository(o.DBRepository),
	)
	needsUpdate, err := client.NeedsUpdate("", o.SkipUpdateDB)
	if err != nil {
		return fmt.Errorf("initDB: client.NeedsUpdate: %w", err)
	}

	if needsUpdate {
		logrus.Info("Updating the trivy vulnerability database...")
		logrus.Infof("Vulnerability database repository: %s", o.DBRepository)
		if err = client.Download(ctx, o.CacheDirectory, ftypes.RegistryOptions{
			Insecure: o.InsecureSkipTLSVerify,
		}); err != nil {
			return fmt.Errorf("failed to download trivy vulnerability DB: %w", err)
		}
	}

	if err := db.Init(o.CacheDirectory); err != nil {
		return fmt.Errorf("failed to init trivy vulnerability DB: %w", err)
	}

	logrus.Debugf("Initialized trivy vulnerability database.")
	return nil
}
