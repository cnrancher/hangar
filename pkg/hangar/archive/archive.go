package archive

import (
	"os"
	"path"
	"runtime"
)

const (
	IndexFileName = "index.json"
	SharedBlobDir = "share"

	DeprecatedArchiveDir   = "saved-image-cache"
	DeprecatedArchiveIndex = "saved-images-list.json"
)

type Format int

const (
	// UNDEFINED is the undefined format
	UNDEFINED Format = iota
	// TAR is the default uncompressed tar archive (tarball)
	TAR
	// GZIP is the gzip format compressed tar archive (tar.gz)
	GZIP
	// ZSTD is the zstd format compressed tar archive (tar.zstd)
	ZSTD
)

func (f Format) String() string {
	switch f {
	case TAR:
		return "tar"
	case GZIP:
		return "gzip"
	case ZSTD:
		return "zstd"
	}
	return ""
}

func (f Format) Suffix() string {
	switch f {
	case TAR:
		return ".tar"
	case GZIP:
		return "tar.gz"
	case ZSTD:
		return ".tar.zstd"
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
