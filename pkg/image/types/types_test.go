package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FilterSet(t *testing.T) {
	var set FilterSet
	set = NewImageFilterSet(nil, nil, nil, false)
	assert.True(t, set.AllowArch("arm64"))
	assert.True(t, set.AllowOS("linux"))
	assert.True(t, set.AllowVariant("v8"))
	assert.True(t, set.AllowVariant(""))

	set = NewImageFilterSet([]string{"arm64", "arm"}, []string{"linux"}, []string{"v8", "v7"}, true)
	assert.True(t, set.AllowArch("arm64"))
	assert.False(t, set.AllowArch("amd64"))
	assert.True(t, set.AllowOS("linux"))
	assert.False(t, set.AllowOS("windows"))
	assert.True(t, set.AllowVariant("v8"))
	assert.True(t, set.AllowVariant(""))
	assert.True(t, set.AllowVariant("v7"))
	assert.False(t, set.AllowVariant("v6"))
	assert.True(t, set.AllowArch("unknown"))
	assert.True(t, set.AllowOS("unknown"))
}
