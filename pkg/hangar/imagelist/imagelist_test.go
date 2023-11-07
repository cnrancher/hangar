package imagelist_test

import (
	"testing"

	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/stretchr/testify/assert"
)

func Test_IsMirrorFormat(t *testing.T) {
	assert := assert.New(t)
	if !assert.True(imagelist.IsMirrorFormat("a b c")) {
		return
	}
	if !assert.True(imagelist.IsMirrorFormat("nginx   mirrored-nginx latest")) {
		return
	}
	if !assert.True(imagelist.IsMirrorFormat("  docker.io/nginx quay.io/user/nginx 1.22  ")) {
		return
	}
	if !assert.False(imagelist.IsMirrorFormat("")) {
		return
	}
	if !assert.False(imagelist.IsMirrorFormat("a b")) {
		return
	}
	if !assert.False(imagelist.IsMirrorFormat("docker.io/nginx quay.io/nginx")) {
		return
	}
	if !assert.False(imagelist.IsMirrorFormat("docker.io/nginx")) {
		return
	}
	if !assert.False(imagelist.IsMirrorFormat("nginx")) {
		return
	}
}

func Test_IsDefaultFormat(t *testing.T) {
	assert := assert.New(t)
	if !assert.True(imagelist.IsDefaultFormat("nginx")) {
		return
	}
	if !assert.True(imagelist.IsDefaultFormat("library/nginx")) {
		return
	}
	if !assert.True(imagelist.IsDefaultFormat("docker.io/nginx")) {
		return
	}
	if !assert.True(imagelist.IsDefaultFormat("docker.io/library/nginx:latest")) {
		return
	}
	if !assert.False(imagelist.IsDefaultFormat("")) {
		return
	}
	if !assert.False(imagelist.IsDefaultFormat("a b")) {
		return
	}
	if !assert.False(imagelist.IsDefaultFormat("a b c")) {
		return
	}
}

func Test_Detect(t *testing.T) {
	assert := assert.New(t)
	if !assert.Equal(imagelist.TypeDefault, imagelist.Detect("docker.io/nginx")) {
		return
	}
	if !assert.Equal(imagelist.TypeDefault, imagelist.Detect("nginx")) {
		return
	}
	if !assert.Equal(imagelist.TypeDefault, imagelist.Detect("library/nginx")) {
		return
	}
	if !assert.Equal(imagelist.TypeDefault, imagelist.Detect("docker.io/library/nginx:1.22")) {
		return
	}
	if !assert.Equal(imagelist.TypeMirror, imagelist.Detect("a b c")) {
		return
	}
	if !assert.Equal(imagelist.TypeMirror, imagelist.Detect(" nginx library/mirrored-nginx 1.22  ")) {
		return
	}
	if !assert.Equal(imagelist.TypeUnknow, imagelist.Detect("docker://docker.io/library/nginx:1.22")) {
		return
	}
}

func Test_GetMirrorSpec(t *testing.T) {
	assert := assert.New(t)
	spec, ok := imagelist.GetMirrorSpec("")
	assert.Nil(spec)
	assert.False(ok)
	spec, ok = imagelist.GetMirrorSpec("a b")
	assert.Nil(spec)
	assert.False(ok)
	spec, ok = imagelist.GetMirrorSpec("a b c")
	assert.NotNil(spec)
	assert.True(ok)
	assert.Equal(3, len(spec))
	assert.Equal("a", spec[0])
	assert.Equal("b", spec[1])
	assert.Equal("c", spec[2])
}
