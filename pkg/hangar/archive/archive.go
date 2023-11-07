package archive

import (
	"os"
	"path"
	"runtime"
)

type Format int

const (
	// UNDEFINED is the undefined format
	UNDEFINED Format = iota
	// TAR is the default uncompressed tar archive
	TAR
	// GZIP is the gzip format compressed tar archive (tar.gz)
	GZIP
)

func (f Format) String() string {
	switch f {
	case TAR:
		return "tar"
	case GZIP:
		return "gzip"
	}
	return ""
}

func (f Format) Suffix() string {
	switch f {
	case TAR:
		return ".tar"
	case GZIP:
		return "tar.gz"
	}
	return ""
}

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
