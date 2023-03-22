package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cnrancher/hangar/pkg/archive"
	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/cnrancher/hangar/pkg/registry"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type loadCmd struct {
	baseCmd

	compressFormat archive.CompressFormat
	directory      string
	mirrorers      []*mirror.Mirror
}

func newLoadCmd() *loadCmd {
	cc := &loadCmd{
		compressFormat: archive.CompressFormatGzip,
		directory:      ".",
	}

	cc.baseCmd.cmd = &cobra.Command{
		Use:     "load",
		Short:   "Load images from saved file into destination registry",
		Long:    `Load images from saved file into destination registry`,
		Example: `  hangar load -s SAVED_FILE.tar.gz -d REGISTRY_URL`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initializeFlagsConfig(cmd, config.DefaultProvider)

			if config.GetBool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
			}

			if err := cc.baseCmd.selfCheckDependencies(checkAll); err != nil {
				return err
			}
			if err := cc.setupFlags(); err != nil {
				return err
			}
			if cc.compressFormat != archive.CompressFormatDirectory {
				if err := cc.baseCmd.prepareImageCacheDirectory(); err != nil {
					return err
				}
			}
			if err := cc.decompressTarball(); err != nil {
				return err
			}
			if err := cc.baseCmd.processDockerLogin(); err != nil {
				return err
			}
			if err := cc.prepareMirrorers(); err != nil {
				return err
			}
			cc.createHarborProject()
			cc.baseCmd.prepareWorker()
			cc.run()
			cc.finish()

			return nil
		},
	}

	cc.cmd.Flags().StringP("source", "s", "", "saved file to load "+
		"(need to use '--compress' to specify the file format if not gzip)")
	cc.cmd.Flags().StringP("destination", "d", "", "destination registry")
	cc.cmd.Flags().StringP("failed", "o", "load-failed.txt", "file name of the load failed image list")
	cc.cmd.Flags().StringP("repo-type", "", "", "repository type, can be 'harbor' or empty")
	cc.cmd.Flags().StringP("compress", "c", "gzip", "compress format, can be 'gzip', 'zstd' or 'dir'")
	cc.cmd.Flags().StringP("default-project", "", "library",
		"project name (also called 'namespace') when destination image project is empty")
	cc.cmd.Flags().IntP("jobs", "j", 1, "worker number, concurrent mode if larger than 1, max 20")
	cc.cmd.Flags().BoolP("harbor-https", "", true, "use https when create harbor project")

	return cc
}

func (cc *loadCmd) setupFlags() error {
	configData := config.DefaultProvider.Get("")
	b, _ := json.MarshalIndent(configData, "", "  ")
	logrus.Debugf("Config: %v", string(b))

	if config.GetString("source") == "" {
		return fmt.Errorf("source file not specified, use '-s' to specify " +
			"the source file to load")
	}
	if config.GetString("destination") == "" && utils.EnvDestRegistry != "" {
		config.Set("destination", utils.EnvDestRegistry)
	}
	if config.GetString("destination") == "" {
		return fmt.Errorf("destination registry URL not specified, " +
			"use '-d' to specify the destination registry URL")
	}
	err := cc.baseCmd.runDockerLogin(config.GetString("destination"))
	if err != nil {
		// output login failed message only
		logrus.Warn(err)
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
		logrus.Warnf("unrecognized compress format %q, set back to gzip",
			config.GetString("compress"))
		cc.compressFormat = archive.CompressFormatGzip
	}

	return nil
}

func (cc *loadCmd) decompressTarball() error {
	cc.directory, _ = utils.GetAbsPath(".")
	src := config.GetString("source")
	if cc.compressFormat != archive.CompressFormatDirectory {
		// decompress input tar.* tarball
		ext := filepath.Ext(src)
		// if parameter filename already have '.part*' extension
		if strings.Contains(ext, "part") {
			logrus.Infof("file name %q contains 'part*' extension", src)
			src = strings.TrimRight(src, ext)
			logrus.Infof("set load file name to %q", src)
		}
		logrus.Infof("decompressing %s...", src)
		err := archive.Decompress(src, cc.directory, cc.compressFormat)
		if err != nil {
			logrus.Fatal(err)
		}
		cc.directory = filepath.Join(cc.directory, utils.CacheImageDirectory)
		logrus.Debugf("decompressed directory: %s", cc.directory)
	} else {
		cc.directory = filepath.Join(cc.directory, src)
	}
	info, err := os.Stat(cc.directory)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", cc.directory)
	}
	return nil
}

func (cc *loadCmd) prepareMirrorers() error {
	var err error
	cc.mirrorers, err = mirror.LoadSavedTemplates(
		cc.directory,
		config.GetString("destination"),
		config.GetString("default-project"))
	if err != nil {
		return err
	}

	return nil
}

func (cc *loadCmd) createHarborProject() {
	repoType := config.GetString("repo-type")
	if repoType != "harbor" {
		return
	}
	logrus.Infof("start creating harbor projects")
	dstProjMap := map[string]bool{}
	for _, m := range cc.mirrorers {
		dstReg := utils.GetRegistryName(m.Destination)
		dstProj := utils.GetProjectName(m.Destination)
		if !dstProjMap[dstProj] {
			url := fmt.Sprintf("%s/api/v2.0/projects", dstReg)
			if config.GetBool("harbor-https") {
				url = "https://" + url
			} else {
				url = "http://" + url
			}
			user, passwd, _ := registry.GetDockerPassword(dstReg)
			err := registry.CreateHarborProject(dstProj, url, user, passwd)
			if err != nil {
				logrus.Errorf("Failed to create harbor project %q: %q",
					dstProj, err)
			}
			dstProjMap[dstProj] = true
		}
	}
}

func (cc *loadCmd) run() {
	for _, m := range cc.mirrorers {
		cc.baseCmd.workerChan <- m
	}
}
