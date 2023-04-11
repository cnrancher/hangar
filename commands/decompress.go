package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cnrancher/hangar/pkg/archive"
	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type decompressCmd struct {
	baseCmd

	compressFormat archive.CompressFormat
}

func newDecompressCmd() *decompressCmd {
	cc := &decompressCmd{}

	cc.baseCmd.cmd = &cobra.Command{
		Use:     "decompress",
		Short:   "Decompress the tarball",
		Long:    ``,
		Example: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, config.DefaultProvider)

			if config.GetBool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
			}

			if err := cc.setupFlags(); err != nil {
				return err
			}

			if err := cc.baseCmd.prepareImageCacheDirectory(); err != nil {
				return err
			}

			if err := cc.run(); err != nil {
				return err
			}

			return nil
		},
	}
	cc.cmd.Flags().StringP("file", "f", "", "file name to be decompressed (required)")
	cc.cmd.Flags().StringP("format", "", "gzip", "compress format (available: 'gzip', 'zstd')")
	// cc.cmd.Flags().StringP("", "", "", "")

	return cc
}

func (cc *decompressCmd) setupFlags() error {
	configData := config.DefaultProvider.Get("")
	b, _ := json.MarshalIndent(configData, "", "  ")
	logrus.Debugf("config: %v", string(b))

	if config.GetString("file") == "" {
		return fmt.Errorf("use '-f' to specify the saved image cache folder")
	}

	cc.compressFormat = archive.CompressFormatGzip
	switch config.GetString("format") {
	case "gzip":
		cc.compressFormat = archive.CompressFormatGzip
	case "zstd":
		cc.compressFormat = archive.CompressFormatZstd
	default:
		logrus.Warnf("unrecognized compress format %q, set to gzip",
			config.GetString("compress"))
		cc.compressFormat = archive.CompressFormatGzip
	}

	return nil
}

func (cc *decompressCmd) run() error {
	// check is valid cache folder
	file := config.GetString("file")
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory", file)
	}

	// decompress input tar.* tarball
	ext := filepath.Ext(file)
	// if parameter filename already have '.part*' extension
	if strings.Contains(ext, "part") {
		logrus.Infof("file name %q contains 'part*' extension", file)
		file = strings.TrimRight(file, ext)
		logrus.Infof("set decompress file name to %q", file)
	}
	logrus.Infof("decompressing %s...", file)
	dir, _ := utils.GetAbsPath(".")
	err = archive.Decompress(file, dir, cc.compressFormat)
	if err != nil {
		logrus.Fatal(err)
	}
	dir = filepath.Join(dir, utils.CacheImageDirectory)
	logrus.Debugf("decompressed directory: %s", dir)
	info, err = os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", dir)
	}
	logrus.Infof("decompressed %q", file)

	return nil
}
