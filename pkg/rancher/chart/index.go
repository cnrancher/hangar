package chart

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/repo"
)

// BuildOrGetIndex builds or get index from local chart repo directory
func BuildOrGetIndex(dir string) (*repo.IndexFile, error) {
	if err := ensureNoSymlinks(dir); err != nil {
		return nil, err
	}

	var (
		existingIndex *repo.IndexFile
		indexPath     = ""
		builtIndex    = repo.NewIndexFile()
	)

	err := filepath.Walk(dir, func(p string, i os.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("%q: %v", p, err)
			return nil
		}
		if i.Name() == "index.yaml" {
			if indexPath == "" || len(p) < len(indexPath) {
				if index, err := repo.LoadIndexFile(p); err == nil {
					existingIndex = index
					indexPath = p
					return filepath.SkipDir
				}
			}
		}
		if !i.IsDir() {
			return nil
		}

		metadata, err := LoadMetadata(p)
		if err != nil {
			return err
		}
		if metadata == nil {
			return nil
		}

		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return fmt.Errorf("building path for chart at %s: %w", dir, err)
		}

		err = builtIndex.MustAdd(metadata, rel, "", "")
		if err != nil {
			logrus.Warn(err)
		}
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}

	if existingIndex != nil {
		return existingIndex, nil
	}

	return builtIndex, nil
}

func ensureNoSymlinks(dir string) error {
	return filepath.Walk(dir, func(p string, i os.FileInfo, err error) error {
		if err != nil || i == nil {
			return err
		}
		if i.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink found at path %s", p)
		}
		return nil
	})
}

// LoadMetadata loads chart metadata from chart directory (not tgz file)
func LoadMetadata(path string) (*chart.Metadata, error) {
	if s, err := os.Stat(path); err == nil && !s.IsDir() {
		return nil, nil
	}

	if ok, err := chartutil.IsChartDir(path); !ok || err != nil {
		return nil, nil
	}

	c, err := loader.LoadDir(path)
	if err != nil {
		return nil, err
	}

	return c.Metadata, nil
}
