package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/cnrancher/hangar/pkg/archive"
	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type saveCmd struct {
	baseCmd

	savedTemplate    *mirror.SavedListTemplate
	compressFormat   archive.CompressFormat
	compressPartSize int
	listSpec         []*saveImageListSpec
	registriesSet    map[string]struct{}
}

func newSaveCmd() *saveCmd {
	cc := &saveCmd{
		registriesSet: make(map[string]struct{}),
		savedTemplate: mirror.NewSavedListTemplate(),
	}

	cc.baseCmd.cmd = &cobra.Command{
		Use:     "save",
		Short:   "Save images from registry server into local file",
		Long:    `Save images from registry server into local file`,
		Example: `  hangar save -f rancher-images.txt -j [WORKER_NUM] -d SAVED_FILE.tar.gz`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize current command config
			initializeFlagsConfig(cmd, config.DefaultProvider)

			if config.GetBool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
			}

			if err := cc.baseCmd.selfCheckDependencies(); err != nil {
				return err
			}
			if err := cc.setupFlags(); err != nil {
				return err
			}
			if err := cc.baseCmd.processSkopeoLogin(); err != nil {
				return err
			}
			if err := cc.processImageList(); err != nil {
				return err
			}
			mu := sync.RWMutex{}
			cc.baseCmd.workerCallback = func(m *mirror.Mirror) error {
				mu.Lock()
				cc.savedTemplate.Append(m.GetSavedImageTemplate())
				mu.Unlock()
				return nil
			}
			cc.baseCmd.prepareWorker()
			if err := cc.prepareImageCacheDirectory(); err != nil {
				logrus.Warn(err)
			}
			if err := cc.run(); err != nil {
				return err
			}
			// waiting for workers
			cc.baseCmd.finish()
			if err := cc.compressTarball(); err != nil {
				return err
			}

			return nil
		},
	}

	cc.cmd.Flags().StringP("file", "f", "", "image list file (the format as same as 'rancher-images.txt')")
	cc.cmd.Flags().StringP("arch", "a", "amd64,arm64", "architecture list of images, separate with ','")
	cc.cmd.Flags().StringP("source", "s", "", "override the source registry defined in image list")
	cc.cmd.Flags().StringP("destination", "d", "",
		"file name of saved images "+
			"\n(can use '--compress' to specify the output file format, default is gzip) "+
			"\n(default \"saved-images.[FORMAT_SUFFIX]\")")
	cc.cmd.Flags().StringP("failed", "o", "save-failed.txt", "file name of the save failed image list")
	cc.cmd.Flags().StringP("compress", "c", "gzip",
		"compress format, can be 'gzip', 'zstd' or 'dir\n"+
			"(set to 'dir' to disable compression, rename the cache directory only)")
	cc.cmd.Flags().BoolP("part", "", false, "enable segment compress")
	cc.cmd.Flags().StringP("part-size", "", "2G",
		"segment part size (number(Bytes), or a string with 'K', 'M', 'G' suffix)")
	cc.cmd.Flags().BoolP("no-arch-failed", "", true,
		"output image name into the failed list if the image arch does not in the arch list "+
			"specified by the '--arch' parameter")
	cc.cmd.Flags().IntP("jobs", "j", 1, "worker number, concurrent mode if larger than 1, max 20")

	return cc
}

func (cc *saveCmd) setupFlags() error {
	configData := config.DefaultProvider.Get("")
	b, _ := json.MarshalIndent(configData, "", "  ")
	logrus.Debugf("config: %v", string(b))

	// command line parameter is prior than env variable
	if config.GetString("source") == "" && utils.EnvSourceRegistry != "" {
		config.Set("source", utils.EnvSourceRegistry)
	}

	cc.compressFormat = archive.CompressFormatGzip
	switch config.GetString("compress") {
	case "gzip":
		cc.compressFormat = archive.CompressFormatGzip
	case "zstd":
		cc.compressFormat = archive.CompressFormatZstd
	case "dir":
		cc.compressFormat = archive.CompressFormatDirectory
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
		logrus.Infof("set destination file name to default %q", d)
		config.Set("destination", d)
	}

	if !strings.Contains(config.GetString("destination"), ".") {
		d := config.GetString("destination")
		switch cc.compressFormat {
		case archive.CompressFormatGzip:
			d += ".tar.gz"
		case archive.CompressFormatZstd:
			d += ".tar.zstd"
		}
		logrus.Infof("set destination file name to %q", d)
		config.Set("destination", d)
	}

	if cc.compressFormat != archive.CompressFormatDirectory {
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
	}

	return nil
}

func (cc *saveCmd) processImageList() error {
	logrus.Debugf("source registry %q", config.GetString("source"))
	fName := config.GetString("file")
	if fName == "" {
		return fmt.Errorf("image list file name not specified")
	}

	f, err := os.Open(fName)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Split(bufio.ScanLines)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			continue
		}
		logrus.Debugf("read line: %v", l)
		spec := newSaveImageListSpec(l)
		if spec == nil {
			logrus.Warnf("ignore line %q in list file: invalid format", l)
			continue
		}
		spec.image = utils.ConstructRegistry(
			spec.image, config.GetString("source"))
		// get the image registry and login
		reg := utils.GetRegistryName(spec.image)
		if _, ok := cc.registriesSet[reg]; !ok {
			cc.registriesSet[reg] = struct{}{}
		}
		cc.listSpec = append(cc.listSpec, spec)
	}

	for r := range cc.registriesSet {
		if err := cc.baseCmd.runSkopeoLogin(r); err != nil {
			// output the login failed message only
			logrus.Warn(err)
		}
	}

	return nil
}

func (cc *saveCmd) run() error {
	for i, v := range cc.listSpec {
		src := utils.ConstructRegistry(v.image, config.GetString("source"))
		if utils.GetProjectName(src) == "" {
			src = utils.ReplaceProjectName(src, "library")
		}
		m := mirror.NewMirror(&mirror.MirrorOptions{
			Source:      src,
			Destination: src,
			Tag:         v.tag,
			Directory:   utils.CacheImageDirectory,
			ArchList:    strings.Split(config.GetString("arch"), ","),
			Line:        v.line,
			Mode:        mirror.MODE_SAVE,
			ID:          i + 1,
		})
		cc.workerChan <- m
	}
	return nil
}

func (cc *saveCmd) compressTarball() error {
	if len(cc.savedTemplate.List) > 0 {
		f := filepath.Join(utils.CacheImageDirectory, utils.SavedImageListFile)
		utils.SaveJson(cc.savedTemplate, f)
	} else {
		logrus.Error("no images saved into local directory, skip.")
		return fmt.Errorf("no images saved")
	}

	if cc.compressFormat != archive.CompressFormatDirectory {
		logrus.Infof("compressing %s...", config.GetString("destination"))

		err := archive.Compress(
			utils.CacheImageDirectory,
			config.GetString("destination"),
			cc.compressFormat,
			cc.compressPartSize,
		)
		if err != nil {
			return err
		}
		if !config.GetBool("part") {
			// if part compress not enabled,
			// rename file name without .part extension
			if err := os.Rename(
				config.GetString("destination")+".part0",
				config.GetString("destination")); err != nil {
				logrus.Warn(err)
			}
		}
	} else {
		err := os.Rename(utils.CacheImageDirectory, config.GetString("destination"))
		if err != nil {
			logrus.Warn(err)
		}
	}
	logrus.Infof("saved images into %q", config.GetString("destination"))
	return nil
}

type saveImageListSpec struct {
	image string
	tag   string
	line  string
}

func newSaveImageListSpec(l string) *saveImageListSpec {
	if strings.Contains(l, " ") {
		return nil
	}

	var v []string = make([]string, 0, 2)
	for _, s := range strings.Split(l, ":") {
		if s != "" {
			v = append(v, s)
		}
	}
	if len(v) != 2 && len(v) != 1 {
		return nil
	}
	// if image name does not have tag, add 'latest'
	if len(v) == 1 {
		v = append(v, "latest")
	}

	return &saveImageListSpec{
		image: v[0],
		tag:   v[1],
		line:  l,
	}
}
