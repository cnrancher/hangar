package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

type syncCmd struct {
	baseCmd

	savedTemplate *mirror.SavedListTemplate
	listSpec      []*saveImageListSpec
	registriesSet map[string]struct{}
}

func newSyncCmd() *syncCmd {
	cc := &syncCmd{
		registriesSet: make(map[string]struct{}),
		savedTemplate: nil,
	}
	cc.baseCmd.cmd = &cobra.Command{
		Use:   "sync",
		Short: "Sync images into decompressed saved images folder",
		Long: `Sync command allows saving extra images into already saved and decompressed folder.

Some images will fail to save with poor network connection; the save failed image list is 'save-failed.txt'.
To add the save failed images into the saved folder:

  hangar sync -f save-failed.txt -d [DECOMPRESSED_FOLDER]

After syncing images into the decompressed folder, you can compress the folder with 'hangar compress' command.`,
		Example: `  hangar sync -f save-failed.txt -d [DECOMPRESSED_FOLDER]`,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			if err := cc.prepareSavedTemplate(); err != nil {
				return err
			}
			if err := cc.processImageList(); err != nil {
				return err
			}
			mu := sync.RWMutex{}
			cc.baseCmd.workerCallback = func(m *mirror.Mirror) error {
				mu.Lock()
				t := m.GetSavedImageTemplate()
				if !cc.savedTemplate.Has(t) {
					cc.savedTemplate.Append(t)
				} else {
					logrus.Debugf(
						"%q already contains in saved template list, skip",
						fmt.Sprintf("%s:%s", t.Source, t.Tag),
					)
				}
				mu.Unlock()
				return nil
			}
			cc.baseCmd.prepareWorker()
			if err := cc.run(); err != nil {
				return err
			}
			cc.baseCmd.finish()
			if err := cc.updateTemplate(); err != nil {
				return err
			}

			return nil
		},
	}
	cc.cmd.Flags().StringP("file", "f", "", "image list file (the format as same as 'rancher-images.txt') (required)")
	cc.cmd.Flags().StringP("arch", "a", "amd64,arm64", "architecture list of images, separate with ','")
	cc.cmd.Flags().StringP("os", "", "linux,windows", "OS list of images, separate with ','")
	cc.cmd.Flags().StringP("source", "s", "", "override the source registry defined in image list")
	cc.cmd.Flags().StringP("destination", "d", "", "decompressed saved images folder (required)")
	cc.cmd.Flags().IntP("jobs", "j", 1, "worker number, concurrent mode if larger than 1, max 20")
	cc.cmd.Flags().StringP("failed", "o", "sync-failed.txt", "file name of the sync failed image list")
	cc.cmd.Flags().BoolP("no-arch-os-fail", "", false,
		"image copy failed when the OS and architecture of the image are not supported")

	return cc
}

func (cc *syncCmd) setupFlags() error {
	configData := config.DefaultProvider.Get("")
	b, _ := json.MarshalIndent(configData, "", "  ")
	logrus.Debugf("config: %v", string(b))

	// command line parameter is prior than env variable
	if config.GetString("source") == "" && utils.EnvSourceRegistry != "" {
		config.Set("source", utils.EnvSourceRegistry)
	}

	if config.GetString("file") == "" {
		return fmt.Errorf("image list file not specified, use '-f' to specify image list file")
	}

	if config.GetString("destination") == "" {
		return fmt.Errorf("saved image folder not specified, use '-d' to specify the saved image folder")
	}

	return nil
}

func (cc *syncCmd) prepareSavedTemplate() error {
	savedList := mirror.SavedListTemplate{}
	dir := config.GetString("destination")
	f, err := os.Open(
		filepath.Join(dir, utils.SavedImageListFile))
	if err != nil {
		return err
	}
	err = json.NewDecoder(f).Decode(&savedList)
	if err != nil {
		return err
	}
	logrus.Debugf("savedList.Version: %v", savedList.Version)
	sVersion := savedList.Version
	if !strings.HasPrefix(sVersion, "v") {
		sVersion = "v" + sVersion
	}
	if semver.Compare(sVersion, mirror.SavedTemplateVersion) != 0 {
		logrus.Warnf("template version in saved file is %q", sVersion)
		logrus.Warnf("the template version supported of this tool is %q",
			mirror.SavedTemplateVersion)
		return fmt.Errorf(
			"this tool does not support template version %q",
			sVersion)
	}
	// reset saved time
	savedList.SavedTime = time.Now().Format(time.RFC3339)
	cc.savedTemplate = &savedList

	return nil
}

func (cc *syncCmd) processImageList() error {
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

func (cc *syncCmd) run() error {
	for i, v := range cc.listSpec {
		src := utils.ConstructRegistry(v.image, config.GetString("source"))
		if utils.GetProjectName(src) == "" {
			src = utils.ReplaceProjectName(src, "library")
		}
		m := mirror.NewMirror(&mirror.MirrorOptions{
			Source:      src,
			Destination: src,
			Tag:         v.tag,
			Directory:   config.GetString("destination"),
			ArchList:    strings.Split(config.GetString("arch"), ","),
			OsList:      strings.Split(config.GetString("os"), ","),
			Line:        v.line,
			Mode:        mirror.MODE_SAVE,
			ID:          i + 1,
		})
		cc.workerChan <- m
	}
	return nil
}

func (cc *syncCmd) updateTemplate() error {
	if len(cc.savedTemplate.List) == 0 {
		logrus.Error("no images saved into local directory, skip.")
		return fmt.Errorf("no images saved")
	}
	f := filepath.Join(config.GetString("destination"),
		utils.SavedImageListFile)
	utils.SaveJson(cc.savedTemplate, f)

	return nil
}
