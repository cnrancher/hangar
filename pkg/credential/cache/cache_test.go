package cache_test

import (
	"testing"

	"github.com/cnrancher/hangar/pkg/credential/cache"
	"github.com/stretchr/testify/assert"
)

func Test_Add_Get_Cached(t *testing.T) {
	cache.Add("admin", "Harbor12345", "harbor.example.io")
	cache.Add("user2", "passwd", "")

	u, p := cache.Get("harbor.example.io")
	assert.Equal(t, u, "admin")
	assert.Equal(t, p, "Harbor12345")

	u, p = cache.Get("")
	assert.Equal(t, u, "user2")
	assert.Equal(t, p, "passwd")

	u, p = cache.Get("unknow.io")
	assert.Equal(t, u, "")
	assert.Equal(t, p, "")

	if t.Failed() {
		return
	}

	assert.True(t, cache.Cached("admin", "Harbor12345", "harbor.example.io"))
	assert.False(t, cache.Cached("user1", "123123", "harbor.example.io"))
	assert.False(t, cache.Cached("unknow", "123123", ""))
}
