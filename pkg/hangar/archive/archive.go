package archive

import (
	"os"
	"path"
	"runtime"
)

const (
	IndexFileName = "index.json"
	SharedBlobDir = "share"
)

var (
	cacheDir string
)

func init() {
	if runtime.GOOS == "darwin" {
		cacheDir = path.Join(os.Getenv("HOME"), ".cache", "hangar_cache")
	} else {
		cacheDir = path.Join(os.Getenv("HOME"), ".cache", "hangar_cache")
	}
	os.MkdirAll(cacheDir, 0755)
}

func CacheDir() string {
	return cacheDir
}
