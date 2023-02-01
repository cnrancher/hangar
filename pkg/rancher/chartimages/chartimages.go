package chartimages

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	u "github.com/cnrancher/image-tools/pkg/utils"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/klauspost/pgzip"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/repo"
)

const RancherVersionAnnotationKey = "catalog.cattle.io/rancher-version"

// chartsToCheckConstraints and *ChartsToCheckConstraints define
// which charts and system charts should be checked for images and added to
// imageSet based on whether the given Rancher version/tag satisfies the chart's
// Rancher version constraints to allow support for multiple version lines of
// a chart in airgap setups. If a chart is not defined here, only the latest
// version of it will be checked for images.
// INFO: CRD charts need to be added as well.
var (
	ChartsToCheckConstraints       = map[string]bool{}
	SystemChartsToCheckConstraints = map[string]bool{
		"rancher-monitoring": true,
	}
)

type OsType int
type ChartRepoType int

const (
	Linux OsType = iota
	Windows
)

const (
	RepoTypeDefault = iota
	RepoTypeSystem
)

func (t *OsType) String() string {
	if t == nil {
		return ""
	}
	switch *t {
	case Linux:
		return "Linux"
	case Windows:
		return "Windows"
	}
	return ""
}

func (t *ChartRepoType) String() string {
	if t == nil {
		return ""
	}
	switch *t {
	case RepoTypeDefault:
		return "default"
	case RepoTypeSystem:
		return "system"
	}
	return ""
}

type Chart struct {
	RancherVersion string
	OS             OsType
	Type           ChartRepoType // chart type: default, system, etc...
	Path           string
	URL            string
	CloneBaseDir   string // directory to clone
	Branch         string // git branch if in URL mode

	ImageSet map[string]map[string]bool // map[image]map[source]
}

type Questions struct {
	RancherMinVersion string `yaml:"rancher_min_version"`
	RancherMaxVersion string `yaml:"rancher_max_version"`
}

func (c *Chart) FetchImages() error {
	if c.ImageSet == nil {
		c.ImageSet = make(map[string]map[string]bool)
	}
	switch {
	case c.Path != "":
		return c.fetchChartsFromPath()
	case c.URL != "":
		return c.fetchChartsFromURL()
	default:
		return fmt.Errorf("chart Path or URL not specified")
	}
}

func (c *Chart) fetchChartsFromPath() error {
	logrus.Infof("Fetching %q chart images from %q",
		c.OS.String(), c.Path)
	index, err := BuildOrGetIndex(c.Path)
	if err != nil {
		return err
	}
	var filteredVersions repo.ChartVersions
	for _, versions := range index.Entries {
		if len(versions) == 0 {
			continue
		}
		// Always append the latest version of the chart.
		latestVersion, latestVersionIndex := pickLatestChartVersion(versions)
		if latestVersionIndex == -1 || latestVersion == nil {
			logrus.Warnf("Failed to get maximum version of chart %q",
				versions[0].Name)
			latestVersion = versions[0]
			latestVersionIndex = 0
		}
		filteredVersions = append(filteredVersions, latestVersion)
		// Append the remaining versions of the chart if the chart exists in
		// the chartsToCheckConstraints map and the given Rancher version
		// satisfies the chart's Rancher version constraint annotation.
		chartName := versions[latestVersionIndex].Name
		var checkConstraints map[string]bool
		switch c.Type {
		case RepoTypeDefault:
			checkConstraints = ChartsToCheckConstraints
		case RepoTypeSystem:
			checkConstraints = SystemChartsToCheckConstraints
		default:
			return fmt.Errorf(
				"fetchChartsFromPath: unrecognized chart type: %v", c.Type)
		}
		if _, ok := checkConstraints[chartName]; ok {
			logrus.Debugf("Check all constraints of chart %q", chartName)
			checkVersions := append(
				versions[0:latestVersionIndex],
				versions[latestVersionIndex+1:]...)
			for _, version := range checkVersions {
				constraint, err := c.checkChartVersionConstraint(*version)
				if err != nil {
					return fmt.Errorf("fetchChartsFromPath: "+
						"failed to check constraint: %w", err)
				}
				if constraint {
					// logrus.Debugf("constraint: %v", version.Version)
					filteredVersions = append(filteredVersions, version)
				}
			}
		}
	}

	// Find values.yaml files of each chart, and check for images
	for _, version := range filteredVersions {
		path := filepath.Join(c.Path, version.URLs[0])
		info, err := os.Stat(path)
		if err != nil {
			logrus.Warn(err)
			continue
		}
		var versionValues []map[interface{}]interface{}
		if info.IsDir() {
			versionValues, err = decodeValuesInDir(path)
		} else {
			versionValues, err = decodeValuesInTgz(path)
		}
		if err != nil {
			logrus.Warn(err)
			continue
		}
		// chartRepoName := filepath.Base(c.Path)
		chartSource := fmt.Sprintf("[%s;%s:%s]",
			c.Path, version.Name, version.Version)
		for _, values := range versionValues {
			err := pickImagesFromValuesMap(
				c.ImageSet, values, chartSource, c.OS)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// fetchChartsFromURL clones the chart git repo into current dir and generate
// image list from it.
func (c *Chart) fetchChartsFromURL() error {
	urlWithoutExt := c.URL
	if strings.HasSuffix(c.URL, ".git") {
		urlWithoutExt = strings.TrimSuffix(c.URL, ".git")
	}
	urlParsed, err := url.Parse(urlWithoutExt)
	if err != nil {
		return fmt.Errorf("fetchChartsFromURL: %w", err)
	}
	option := git.CloneOptions{
		URL:               c.URL,
		RecurseSubmodules: git.NoRecurseSubmodules,
		Depth:             1,
	}
	if c.Branch != "" {
		option.ReferenceName = plumbing.NewBranchReferenceName(c.Branch)
	}
	directory := filepath.Join(
		c.CloneBaseDir, strings.TrimLeft(urlParsed.Path, "/"))
	logrus.Infof("Cloning git repo into %q, branch %q",
		directory, c.Branch)
	r, err := git.PlainClone(directory, false, &option)
	if err != nil {
		if !errors.Is(err, git.ErrRepositoryAlreadyExists) {
			return fmt.Errorf("fetchChartsFromURL: %w", err)
		}
	}
	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		logrus.Infof("Git repo %q already exists", directory)
		r, err = git.PlainOpen(directory)
		if err != nil {
			return fmt.Errorf("fetchChartsFromURL: %w", err)
		}
	}
	// remotes, err := r.Remotes()
	// if err != nil {
	// 	return fmt.Errorf("fetchChartsFromURL: remotes:  %w", err)
	// }
	// if len(remotes) == 0 {
	// 	return fmt.Errorf("fetchChartsFromURL: remote is empty")
	// }
	// err = r.Fetch(&git.FetchOptions{
	// 	RefSpecs:   []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
	// 	RemoteName: remotes[0].Config().Name,
	// })
	// if err != nil {
	// 	return fmt.Errorf("fetchChartsFromURL: fetch: %w", err)
	// }
	worktree, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("fetchChartsFromURL: worktree: %w", err)
	}
	worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(c.Branch),
	})
	if err != nil {
		return fmt.Errorf("fetchChartsFromURL: checkout: %w", err)
	}
	c.Path = directory

	return c.fetchChartsFromPath()
}

// checkChartVersionConstraint retrieves the value of a chart's rancher-version
// annotation, or rancher_min/max_version in questions.yaml and returns true
// if the rancher-version in the export configuration satisfies the chart's
// constraint, false otherwise.
func (c Chart) checkChartVersionConstraint(
	version repo.ChartVersion,
) (bool, error) {
	constraintStr, ok := version.Annotations[RancherVersionAnnotationKey]
	if ok {
		return compareRancherVersionToConstraint(
			c.RancherVersion, constraintStr)
	}
	// if no rancher version annotation, check questions.yaml file
	questionsPath := filepath.Join(
		c.Path, version.URLs[0], "questions.yaml")
	questions, err := decodeQuestionsFile(questionsPath)
	if os.IsNotExist(err) {
		questionsPath = filepath.Join(
			c.Path, version.URLs[0], "questions.yml")
		questions, err = decodeQuestionsFile(questionsPath)
	}
	if err != nil {
		logrus.Warnf("Skip %s:%s does not have a questions file",
			version.Name, version.Version)
		return false, nil
	}
	constraintStr = minMaxToConstraintStr(
		questions.RancherMinVersion, questions.RancherMaxVersion)
	if constraintStr == "" {
		logrus.Warnf("Failed to read RancherMin/MaxVersion from %s:%s",
			version.Name, version.Version)
		return false, nil
	}
	return compareRancherVersionToConstraint(
		c.RancherVersion, constraintStr)
}

func decodeQuestionsFile(path string) (Questions, error) {
	var questions Questions
	file, err := os.Open(path)
	if err != nil {
		return Questions{}, err
	}
	defer file.Close()
	if err := decodeYAMLFile(file, &questions); err != nil {
		return Questions{}, err
	}
	return questions, nil
}

// minMaxToConstraintStr converts min and max Rancher version strings into a
// constraint string
// E.g min "2.6.3" max "2.6.4" -> constraintStr "2.6.3 - 2.6.4".
func minMaxToConstraintStr(min, max string) string {
	if min != "" && max != "" {
		return fmt.Sprintf("%s - %s", min, max)
	}
	if min != "" {
		return fmt.Sprintf(">= %s", min)
	}
	if max != "" {
		return fmt.Sprintf("<= %s", max)
	}
	return ""
}

// compareRancherVersionToConstraint returns true if the rancher-version
// satisfies constraintStr, false otherwise.
func compareRancherVersionToConstraint(
	rancherVersion, constraintStr string,
) (bool, error) {
	if constraintStr == "" {
		return false, fmt.Errorf("constraint is empty string")
	}
	c, err := semver.NewConstraint(constraintStr)
	if err != nil {
		return false, err
	}
	rancherSemVer, err := semver.NewVersion(rancherVersion)
	if err != nil {
		return false, err
	}
	// When the exporter is ran in a dev environment, we replace
	// the rancher version with a dev version (e.g 2.X.99).
	// This breaks the semver compare logic for exporting because
	// we use the Rancher version constraint < 2.X.99-0 in
	// many of our charts and since 2.X.99 > 2.X.99-0 the comparison
	// returns false which is not the desired behavior.
	patch := rancherSemVer.Patch()
	if patch == 99 {
		patch = 98
	}
	// All pre-release versions are removed because the semver
	// comparison will not yield the desired behavior unless
	// the constraint has a pre-release too. Since the exporter
	// for charts can treat pre-releases and releases equally,
	// is cleaner to remove it. E.g. comparing rancherVersion
	// 2.6.4-rc1 and constraint 2.6.3 - 2.6.5 yields false because
	// the versions in the contraint do not have a pre-release.
	// This behavior comes from the semver module and is intentional.
	rSemVer, err := semver.NewVersion(fmt.Sprintf("%d.%d.%d",
		rancherSemVer.Major(), rancherSemVer.Minor(), patch))
	if err != nil {
		return false, err
	}
	return c.Check(rSemVer), nil
}

// pickImagesFromValuesMap walks a values map to find images,
// and add them to imagesSet.
func pickImagesFromValuesMap(
	imagesSet map[string]map[string]bool,
	values map[interface{}]interface{},
	chartSource string,
	OS OsType,
) error {
	walkMap(values, func(inputMap map[interface{}]interface{}) {
		repository, ok := inputMap["repository"].(string)
		if !ok {
			return
		}
		// No string type assertion because some charts
		// have float typed image tags
		tag, ok := inputMap["tag"]
		if !ok {
			return
		}
		imageName := fmt.Sprintf("%s:%v", repository, tag)
		// By default, images are added to the generic images list ("linux").
		// For Windows and multi-OS images to be considered, they must use a
		// comma-delineated list (e.g. "os: windows", "os: windows,linux",
		// and "os: linux,windows").
		osList, ok := inputMap["os"].(string)
		if !ok {
			if inputMap["os"] != nil {
				logrus.Errorf(
					"field 'os:' for image %s neither a string nor nil",
					imageName)
			}
			if OS == Linux {
				u.AddSourceToImage(imagesSet, imageName, chartSource)
				return
			}
		}
		for _, os := range strings.Split(osList, ",") {
			os = strings.TrimSpace(os)
			if strings.EqualFold("windows", os) && OS == Windows {
				u.AddSourceToImage(imagesSet, imageName, chartSource)
				return
			}
			if strings.EqualFold("linux", os) && OS == Linux {
				u.AddSourceToImage(imagesSet, imageName, chartSource)
				return
			}
		}
	})
	return nil
}

// walkMap walks inputMap and calls the callback function on all map
// type nodes including the root node.
func walkMap(inputMap interface{}, cb func(map[interface{}]interface{})) {
	switch data := inputMap.(type) {
	case map[interface{}]interface{}:
		cb(data)
		for _, value := range data {
			walkMap(value, cb)
		}
	case []interface{}:
		for _, elem := range data {
			walkMap(elem, cb)
		}
	}
}

// decodeValuesInTgz reads tarball and returns a slice of values
// corresponding to values.yaml files found inside of it.
func decodeValuesInTgz(path string) ([]map[interface{}]interface{}, error) {
	tgz, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer tgz.Close()
	gzr, err := pgzip.NewReader(tgz)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	var valuesSlice []map[interface{}]interface{}
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return valuesSlice, nil
		case err != nil:
			return nil, err
		case header.Typeflag == tar.TypeReg && isValuesFile(header.Name):
			var values map[interface{}]interface{}
			if err := decodeYAMLFile(tr, &values); err != nil {
				return nil, err
			}
			valuesSlice = append(valuesSlice, values)
		default:
			continue
		}
	}
}

// decodeValuesInDir reads directory and returns a slice of values
// corresponding to values.yaml files found inside of it.
func decodeValuesInDir(dir string) ([]map[interface{}]interface{}, error) {
	var valuesSlice []map[interface{}]interface{}
	err := filepath.Walk(dir, func(p string, i fs.FileInfo, err error) error {
		if err != nil {
			logrus.Warn(err)
			return nil
		}
		if i.IsDir() {
			return nil
		}
		if isValuesFile(i.Name()) {
			var values map[interface{}]interface{}
			f, err := os.Open(p)
			if err != nil {
				logrus.Warn(err)
				return nil
			}
			if err := decodeYAMLFile(f, &values); err != nil {
				return err
			}
			valuesSlice = append(valuesSlice, values)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return valuesSlice, nil
}

func isValuesFile(path string) bool {
	basename := filepath.Base(path)
	return basename == "values.yaml" || basename == "values.yml"
}

func decodeYAMLFile(r io.Reader, target interface{}) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}

// pickLatestChartVersion finds the latest version of charts from version list
func pickLatestChartVersion(vs repo.ChartVersions) (*repo.ChartVersion, int) {
	if len(vs) == 0 {
		return nil, -1
	}
	maximumVersion := vs[0].Version
	maximumIndex := -1
	for i := range vs {
		currVersion, err := semver.NewVersion(vs[i].Version)
		if err != nil {
			return nil, -1
		}
		maxVersion, err := semver.NewVersion(maximumVersion)
		if err != nil {
			return nil, -1
		}
		if currVersion.Compare(maxVersion) >= 0 {
			maximumVersion = vs[i].Version
			maximumIndex = i
		}
	}
	return vs[maximumIndex], maximumIndex
}
