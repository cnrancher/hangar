package kdmimages

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_filterDeprecatedVersions(t *testing.T) {
	testVersions := []string{
		"v1.21.0",
		"v1.21.1",
		"v1.21.2",
	}
	v := filterDeprecatedVersions(testVersions)
	if !assert.Equal(t, []string{"v1.21.2"}, v) {
		return
	}
	t.Logf("filterDeprecatedVersions: %v", v)

	testVersions = []string{
		"v1.21.0",
		"v1.21.1",
		"v1.21.2",
		"v1.22.1",
		"v1.22.2",
		"v1.28.2",
		"v1.28.3",
	}
	v = filterDeprecatedVersions(testVersions)
	if !assert.Equal(t, []string{"v1.21.2", "v1.22.2", "v1.28.3"}, v) {
		return
	}
	t.Logf("filterDeprecatedVersions: %v", v)

	v = filterDeprecatedVersions([]string{})
	if !assert.Equal(t, []string{}, v) {
		return
	}
	t.Logf("filterDeprecatedVersions: %v", v)
}
