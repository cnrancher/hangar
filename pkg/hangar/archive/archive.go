package archive

import (
	"os"
	"path"
)

const (
	IndexFileName = "index.json"
	SharedBlobDir = "share"
)

var (
	cacheDir string
)

func init() {
	if os.Getenv("HOME") == "" {
		// Use /var/tmp/hangar_cache as cache folder.
		cacheDir = path.Join("/", "var", "tmp", "hangar_cache")
	} else {
		// Use ${HOME}/.cache/hangar_cache as cache folder
		cacheDir = path.Join(os.Getenv("HOME"), ".cache", "hangar_cache")
	}
	os.MkdirAll(cacheDir, 0755)
}

func CacheDir() string {
	return cacheDir
}
