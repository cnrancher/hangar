package kdmimages

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_filterDeprecatedVersions(t *testing.T) {
	testVersions := []string{
		"v1.21.0+rke2r1",
		"v1.21.1+rke2r1",
		"v1.21.2+rke2r1",
	}
	v := filterDeprecatedVersions(testVersions)
	sort.Strings(v)
	if !assert.Equal(t, []string{"v1.21.2+rke2r1"}, v) {
		return
	}
	t.Logf("filterDeprecatedVersions: %v", v)

	testVersions = []string{
		"v1.21.0+rke2r1",
		"v1.21.1+rke2r1",
		"v1.21.2+rke2r1",
		"v1.22.1+rke2r1",
		"v1.22.2+rke2r1",
		"v1.28.2+rke2r1",
		"v1.28.3+rke2r1",
		"v1.28.3+rke2r2",
	}
	v = filterDeprecatedVersions(testVersions)
	sort.Strings(v)
	if !assert.Equal(t, []string{
		"v1.21.2+rke2r1",
		"v1.22.2+rke2r1",
		"v1.28.3+rke2r2"}, v) {
		return
	}
	t.Logf("filterDeprecatedVersions: %v", v)

	v = filterDeprecatedVersions([]string{})
	sort.Strings(v)
	if !assert.Equal(t, []string{}, v) {
		return
	}
	t.Logf("filterDeprecatedVersions: %v", v)
}
