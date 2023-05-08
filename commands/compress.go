package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cnrancher/hangar/pkg/archive"
	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type compressCmd struct {
	baseCmd

	compressFormat   archive.CompressFormat
	compressPartSize int
}

func newCompressCmd() *compressCmd {
	cc := &compressCmd{}

	cc.baseCmd.cmd = &cobra.Command{
		Use:     "compress",
		Short:   "Compress the saved image cache folder",
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

			if err := cc.run(); err != nil {
				return err
			}

			return nil
		},
	}
	cc.cmd.Flags().StringP("file", "f", "", "saved image cache folder (required)")
	cc.cmd.Flags().StringP("destination", "d", "",
		"file name of saved images "+
			"\n(can use '--compress' to specify the output file format, default is gzip) "+
			"\n(default \"saved-images.[FORMAT_SUFFIX]\")")
	cc.cmd.Flags().StringP("format", "", "gzip", "compress format (available: 'gzip', 'zstd')")
	cc.cmd.Flags().BoolP("part", "", false, "enable segment compress")
	cc.cmd.Flags().StringP("part-size", "", "2G",
		"segment part size (number(Bytes), or a string with 'K', 'M', 'G' suffix)")

	return cc
}

func (cc *compressCmd) setupFlags() error {
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

	if config.GetString("destination") == "" {
		d := "saved-images"
		switch cc.compressFormat {
		case archive.CompressFormatGzip:
			d += ".tar.gz"
		case archive.CompressFormatZstd:
			d += ".tar.zstd"
		}
		logrus.Debugf("set destination file name to default %q", d)
		config.Set("destination", d)
	}

	if config.GetBool("part") {
		// segment compress enabled
		sPartSize := config.GetString("part-size")
		var err error
		switch {
		case strings.HasSuffix(sPartSize, "G"):
			cc.compressPartSize, err = strconv.Atoi(
				strings.TrimSuffix(sPartSize, "G"))
			cc.compressPartSize *= archive.GB
		case strings.HasSuffix(sPartSize, "g"):
			cc.compressPartSize, err = strconv.Atoi(
				strings.TrimSuffix(sPartSize, "g"))
			cc.compressPartSize *= archive.GB
		case strings.HasSuffix(sPartSize, "M"):
			cc.compressPartSize, err = strconv.Atoi(
				strings.TrimSuffix(sPartSize, "M"))
			cc.compressPartSize *= archive.MB
		case strings.HasSuffix(sPartSize, "m"):
			cc.compressPartSize, err = strconv.Atoi(
				strings.TrimSuffix(sPartSize, "m"))
			cc.compressPartSize *= archive.MB
		case strings.HasSuffix(sPartSize, "K"):
			cc.compressPartSize, err = strconv.Atoi(
				strings.TrimSuffix(sPartSize, "K"))
			cc.compressPartSize *= archive.KB
		case strings.HasSuffix(sPartSize, "k"):
			cc.compressPartSize, err = strconv.Atoi(
				strings.TrimSuffix(sPartSize, "k"))
			cc.compressPartSize *= archive.KB
		default:
			cc.compressPartSize, err = strconv.Atoi(sPartSize)
		}
		if err != nil {
			return fmt.Errorf("failed to get segment part size: %v", err)
		}
		logrus.Infof("set compress segment part to %q", sPartSize)
	}
	return nil
}

func (cc *compressCmd) run() error {
	// check is valid cache folder
	directory := config.GetString("file")
	directory, err := utils.GetAbsPath(directory)
	if err != nil {
		return fmt.Errorf("utils.GetAbsPath: %w", err)
	}
	info, err := os.Stat(directory)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", directory)
	}

	dirSpecs := strings.Split(directory, string(filepath.Separator))
	if len(dirSpecs) == 0 {
		return fmt.Errorf("invalid directory path: %q", directory)
	}
	if dirSpecs[len(dirSpecs)-1] != utils.CacheImageDirectory {
		dirSpecs[len(dirSpecs)-1] = utils.CacheImageDirectory
		oldDir := directory
		directory = string(filepath.Separator) + filepath.Join(dirSpecs...)
		logrus.Warnf("rename folder %q to %q", oldDir, directory)
		err := os.Rename(oldDir, directory)
		if err != nil {
			return err
		}
	}
	logrus.Debugf("cache folder: %q", directory)

	template := filepath.Join(directory, utils.SavedImageListFile)
	info, err = os.Stat(template)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory", template)
	}

	logrus.Infof("compressing %s...", config.GetString("destination"))

	destination := config.GetString("destination")
	destination, err = utils.GetAbsPath(destination)
	if err != nil {
		return fmt.Errorf("utils.GetAbsPath: %w", err)
	}
	err = os.Chdir(filepath.Join(directory, ".."))
	if err != nil {
		return fmt.Errorf("os.Chdir: %w", err)
	}
	err = archive.Compress(
		// set to compress the cache folder only
		utils.CacheImageDirectory,
		// output the compressed file to current dir
		destination,
		cc.compressFormat,
		cc.compressPartSize,
	)
	if err != nil {
		return err
	}
	if !config.GetBool("part") {
		// if part compress not enabled,
		// rename file name without .part extension
		if err := os.Rename(destination+".part0", destination); err != nil {
			logrus.Warnf("os.Rename: %v", err)
		}
	}
	logrus.Infof("compressed %q", config.GetString("destination"))

	return nil
}
