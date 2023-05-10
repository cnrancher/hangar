package chartimages

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	k8sYaml "sigs.k8s.io/yaml"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

const ChartfileName = "Chart.yaml"

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
				if index, err := repo.LoadIndexFile(p); err != nil {
					logrus.Warnf("Failed to load %q: %v", p, err)
				} else {
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
			logrus.Warnf("failed to add %q into index file: %v", rel, err)
		}
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}

	if existingIndex != nil {
		return existingIndex, nil
	}

	// sort index versions in descending order.
	builtIndex.SortEntries()

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

	if ok, err := IsChartDir(path); !ok || err != nil {
		return nil, nil
	}

	chartYaml := filepath.Join(path, ChartfileName)
	d, err := os.ReadFile(chartYaml)
	if err != nil {
		return nil, fmt.Errorf("LoadMetadata: %w", err)
	}
	metadata := new(chart.Metadata)
	if err := k8sYaml.Unmarshal(d, metadata); err != nil {
		return metadata, fmt.Errorf("cannot load Chart.yaml: %w", err)
	}
	if metadata.APIVersion == "" {
		metadata.APIVersion = "v1"
	}

	return metadata, nil
}

// IsChartDir validate a chart directory.
//
// Checks for a valid Chart.yaml.
func IsChartDir(dirName string) (bool, error) {
	if fi, err := os.Stat(dirName); err != nil {
		return false, err
	} else if !fi.IsDir() {
		return false, fmt.Errorf("%q is not a directory", dirName)
	}

	chartYaml := filepath.Join(dirName, ChartfileName)
	if _, err := os.Stat(chartYaml); os.IsNotExist(err) {
		return false, fmt.Errorf(
			"no %s exists in directory %q", ChartfileName, dirName)
	}

	chartYamlContent, err := os.ReadFile(chartYaml)
	if err != nil {
		return false, fmt.Errorf(
			"cant read %s in directory %q", "Chart", dirName)
	}

	chartContent := new(chart.Metadata)
	if err := k8sYaml.Unmarshal(chartYamlContent, &chartContent); err != nil {
		return false, err
	}
	if chartContent == nil {
		return false, fmt.Errorf("chart metadata (%s) missing", ChartfileName)
	}
	if chartContent.Name == "" {
		return false, fmt.Errorf(
			"invalid chart (%s): name must not be empty", ChartfileName)
	}

	return true, nil
}
