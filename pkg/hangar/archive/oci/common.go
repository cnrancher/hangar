package oci

import (
	"fmt"
	"os"

	"github.com/cnrancher/hangar/pkg/utils"
)

func newFileCacheDir() (string, error) {
	cd, err := os.MkdirTemp(utils.HangarCacheDir(), "*")
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	return cd, nil
}
