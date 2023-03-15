package chartimages

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ChartVersion, IndexFile, Metadata and other chart related data types were
// copied from https://github.com/helm/helm

type ChartVersions []*ChartVersion

// Len returns the length.
func (c ChartVersions) Len() int { return len(c) }

// Swap swaps the position of two items in the versions slice.
func (c ChartVersions) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

// Less returns true if the version of entry a is less than the version of entry b.
func (c ChartVersions) Less(a, b int) bool {
	// Failed parse pushes to the back.
	i, err := semver.NewVersion(c[a].Version)
	if err != nil {
		return true
	}
	j, err := semver.NewVersion(c[b].Version)
	if err != nil {
		return false
	}
	return i.LessThan(j)
}

type IndexFile struct {
	ServerInfo map[string]interface{}   `json:"serverInfo,omitempty"`
	APIVersion string                   `json:"apiVersion"`
	Generated  time.Time                `json:"generated"`
	Entries    map[string]ChartVersions `json:"entries"`
	PublicKeys []string                 `json:"publicKeys,omitempty"`

	// Annotations are additional mappings uninterpreted by Helm.
	// They are made available for other applications
	// to add information to the index file.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// MustAdd adds a file to the index
// This can leave the index in an unsorted state
func (i IndexFile) MustAdd(
	md *Metadata,
	filename, baseURL, digest string,
) error {
	if i.Entries == nil {
		return errors.New("entries not initialized")
	}

	if md.APIVersion == "" {
		md.APIVersion = "v1"
	}
	// if err := md.Validate(); err != nil {
	// 	return errors.Wrapf(err, "validate failed for %s", filename)
	// }

	u := filename
	if baseURL != "" {
		_, file := filepath.Split(filename)
		var err error
		u, err = url.JoinPath(baseURL, file)
		if err != nil {
			u = path.Join(baseURL, file)
		}
	}
	cr := &ChartVersion{
		URLs:     []string{u},
		Metadata: md,
		Digest:   digest,
		Created:  time.Now(),
	}
	ee := i.Entries[md.Name]
	i.Entries[md.Name] = append(ee, cr)
	return nil
}

// Metadata for a Chart file. This models the structure of a Chart.yaml file.
type Metadata struct {
	// The name of the chart. Required.
	Name string `json:"name,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	Home string `json:"home,omitempty"`
	// Source is the URL to the source code of this chart
	Sources []string `json:"sources,omitempty"`
	// A SemVer 2 conformant version string of the chart. Required.
	Version string `json:"version,omitempty"`
	// A one-sentence description of the chart
	Description string `json:"description,omitempty"`
	// A list of string keywords
	Keywords []string `json:"keywords,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	// Maintainers []*Maintainer `json:"maintainers,omitempty"`
	// The URL to an icon file.
	Icon string `json:"icon,omitempty"`
	// The API Version of this chart. Required.
	APIVersion string `json:"apiVersion,omitempty"`
	// The condition to check to enable chart
	Condition string `json:"condition,omitempty"`
	// The tags to check to enable chart
	Tags string `json:"tags,omitempty"`
	// The version of the application enclosed inside of this chart.
	AppVersion string `json:"appVersion,omitempty"`
	// Whether or not this chart is deprecated
	Deprecated bool `json:"deprecated,omitempty"`
	// Annotations are additional mappings uninterpreted by Helm,
	// made available for inspection by other applications.
	Annotations map[string]string `json:"annotations,omitempty"`
	// KubeVersion is a SemVer constraint
	// specifying the version of Kubernetes required.
	KubeVersion string `json:"kubeVersion,omitempty"`
	// Dependencies are a list of dependencies for a chart.
	// Dependencies []*Dependency `json:"dependencies,omitempty"`
	// Specifies the chart type: application or library
	Type string `json:"type,omitempty"`
}

// ChartVersion represents a chart entry in the IndexFile
type ChartVersion struct {
	*Metadata
	URLs    []string  `json:"urls"`
	Created time.Time `json:"created,omitempty"`
	Removed bool      `json:"removed,omitempty"`
	Digest  string    `json:"digest,omitempty"`

	// ChecksumDeprecated is deprecated in Helm 3, and therefore ignored.
	// Helm 3 replaced this with Digest.
	// However, with a strict YAML parser enabled, a field must be
	// present on the struct for backwards compatibility.
	ChecksumDeprecated string `json:"checksum,omitempty"`

	// EngineDeprecated is deprecated in Helm 3, and therefore ignored.
	// However, with a strict
	// YAML parser enabled, this field must be present.
	EngineDeprecated string `json:"engine,omitempty"`

	// TillerVersionDeprecated is deprecated in Helm 3, and therefore ignored.
	// However, with a strict
	// YAML parser enabled, this field must be present.
	TillerVersionDeprecated string `json:"tillerVersion,omitempty"`

	// URLDeprecated is deprecated in Helm 3, superseded by URLs.
	// It is ignored. However,
	// with a strict YAML parser enabled, this must be present on the struct.
	URLDeprecated string `json:"url,omitempty"`
}

// BuildOrGetIndex builds or get index from local chart repo directory
func BuildOrGetIndex(dir string) (*IndexFile, error) {
	if err := ensureNoSymlinks(dir); err != nil {
		return nil, err
	}

	var (
		existingIndex *IndexFile
		indexPath     = ""
		builtIndex    = NewIndexFile()
	)

	err := filepath.Walk(dir, func(p string, i os.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("%q: %v", p, err)
			return nil
		}
		if i.Name() == "index.yaml" {
			if indexPath == "" || len(p) < len(indexPath) {
				if index, err := LoadIndexFile(p); err == nil {
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
func LoadMetadata(path string) (*Metadata, error) {
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
	metadata := new(Metadata)
	if err := yaml.Unmarshal(d, metadata); err != nil {
		return metadata, fmt.Errorf("cannot load Chart.yaml: %w", err)
	}
	if metadata.APIVersion == "" {
		metadata.APIVersion = "v1"
	}

	return metadata, nil
}

// NewIndexFile initializes an index.
func NewIndexFile() *IndexFile {
	return &IndexFile{
		APIVersion: "v1",
		Generated:  time.Now(),
		Entries:    map[string]ChartVersions{},
		PublicKeys: []string{},
	}
}

// LoadIndexFile takes a file at the given path and returns an IndexFile object
func LoadIndexFile(path string) (*IndexFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	i, err := loadIndex(b, path)
	if err != nil {
		return nil, fmt.Errorf("error loading %s: %w", path, err)
	}
	return i, nil
}

// loadIndex loads an index file and does minimal validity checking.
//
// The source parameter is only used for logging.
// This will fail if API Version is not set or unmarshal fails.
func loadIndex(data []byte, source string) (*IndexFile, error) {
	i := &IndexFile{}

	if len(data) == 0 {
		return i, fmt.Errorf("empty index.yaml file")
	}

	if err := yaml.UnmarshalStrict(data, i); err != nil {
		return i, err
	}

	for name, cvs := range i.Entries {
		for idx := len(cvs) - 1; idx >= 0; idx-- {
			if cvs[idx] == nil {
				logrus.Warnf("skipping loading invalid entry for chart "+
					"%q from %s: empty entry", name, source)
				continue
			}
			if cvs[idx].APIVersion == "" {
				cvs[idx].APIVersion = "v1"
			}
		}
	}
	i.SortEntries()
	if i.APIVersion == "" {
		return i, fmt.Errorf("no API version specified")
	}
	return i, nil
}

func (i IndexFile) SortEntries() {
	for _, versions := range i.Entries {
		sort.Sort(sort.Reverse(versions))
	}
}

const ChartfileName = "Chart.yaml"

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

	chartContent := new(Metadata)
	if err := yaml.Unmarshal(chartYamlContent, &chartContent); err != nil {
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
