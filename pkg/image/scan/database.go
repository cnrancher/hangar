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
	"github.com/google/go-containerregistry/pkg/name"
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
	DBRepositories        []string
	JavaDBRepositories    []string
	SkipUpdateDB          bool
	SkipUpdateJavaDB      bool
	InsecureSkipTLSVerify bool
}

func InitTrivyDatabase(ctx context.Context, o DBOptions) error {
	if dbInitialized {
		return nil
	}

	if len(o.DBRepositories) == 0 {
		o.DBRepositories = []string{
			DefaultECRRepository,
			DefaultGHCRRepository,
		}
	}
	if len(o.JavaDBRepositories) == 0 {
		o.JavaDBRepositories = []string{
			DefaultJavaECRRepository,
			DefaultJavaGHCRRepository,
		}
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
	if len(o.JavaDBRepositories) == 0 {
		return fmt.Errorf("java db repo not specified")
	}
	dbRefs := []name.Reference{}
	for _, r := range o.JavaDBRepositories {
		dbRef, err := name.ParseReference(r)
		if err != nil {
			return fmt.Errorf("failed to parse %q: %w", r, err)
		}
		dbRefs = append(dbRefs, dbRef)
	}
	javadb.Init(
		o.CacheDirectory,
		dbRefs,
		false, false,
		ftypes.RegistryOptions{
			Insecure: o.InsecureSkipTLSVerify,
		},
	)

	dbDir := filepath.Join(o.CacheDirectory, "java-db")
	metac := trivyjavadb.NewMetadata(dbDir)
	meta, err := metac.Get()
	if err != nil {
		logrus.Debugf("java db metadata get error: %v", err)
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("java DB metadata error: %w", err)
		} else if o.SkipUpdateJavaDB {
			logrus.Error("The first run cannot skip downloading Java DB")
			return fmt.Errorf("java database cannot skip on first run")
		}
	}
	needUpdate := false
	now := time.Now().UTC()
	if meta.Version != trivyjavadb.SchemaVersion || meta.NextUpdate.Before(now) {
		needUpdate = true
	}
	if now.Before(meta.DownloadedAt.Add(time.Hour)) || o.SkipUpdateJavaDB {
		needUpdate = false
	}
	if needUpdate {
		// Download DB
		logrus.Infof("Java DB Repositories: %v", o.JavaDBRepositories)
		logrus.Infof("Downloading the Java DB to the cache dir %q", dbDir)

		artifacts := oci.NewArtifacts(dbRefs, ftypes.RegistryOptions{
			Insecure: o.InsecureSkipTLSVerify,
		})
		err = artifacts.Download(ctx, dbDir, oci.DownloadOption{MediaType: mediaType})
		if err != nil {
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
	if len(o.DBRepositories) == 0 {
		return fmt.Errorf("trivy db repository not specified")
	}
	dbRefs := []name.Reference{}
	for _, r := range o.DBRepositories {
		dbRef, err := name.ParseReference(r)
		if err != nil {
			return fmt.Errorf("failed to parse %q: %w", r, err)
		}
		dbRefs = append(dbRefs, dbRef)
	}
	client := trivydb.NewClient(
		o.CacheDirectory, false, trivydb.WithDBRepository(dbRefs),
	)
	needsUpdate, err := client.NeedsUpdate(ctx, "", o.SkipUpdateDB)
	if err != nil {
		return fmt.Errorf("initDB: client.NeedsUpdate: %w", err)
	}

	if needsUpdate {
		logrus.Info("Updating the trivy vulnerability database...")
		logrus.Infof("Vulnerability database repositories: %v", o.DBRepositories)
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
