package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/cnrancher/hangar/pkg/registry"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type mirrorCmd struct {
	baseCmd

	listSpec      []*mirrorImageListSpec
	registriesSet map[string]struct{}
}

func newMirrorCmd() *mirrorCmd {
	cc := &mirrorCmd{
		listSpec:      make([]*mirrorImageListSpec, 0),
		registriesSet: make(map[string]struct{}),
	}

	cc.baseCmd.cmd = &cobra.Command{
		Use:     "mirror",
		Short:   "Mirror images from source registry to destination registry",
		Long:    `Mirror images from source registry to destination registry`,
		Example: "  hangar mirror -f MIRROR_IMAGE_LIST.txt -s SOURCE_REGISTRY -d DEST_REGISTRY",
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
			if err := cc.baseCmd.processDockerLogin(); err != nil {
				return err
			}
			if err := cc.processImageList(); err != nil {
				return err
			}
			cc.createHarborProject()
			cc.baseCmd.prepareWorker()
			cc.run()
			cc.baseCmd.finish()

			return nil
		},
	}

	cc.cmd.Flags().StringP("file", "f", "", "image list file (should be 'mirror' format)")
	cc.cmd.Flags().StringP("arch", "a", "amd64,arm64", "architecture list of images, separate with ','")
	cc.cmd.Flags().StringP("source", "s", "", "override the source registry defined in image list")
	cc.cmd.Flags().StringP("destination", "d", "", "override the destination registry defined in image list")
	cc.cmd.Flags().StringP("failed", "o", "mirror-failed.txt", "file name of the mirror failed image list")
	cc.cmd.Flags().IntP("jobs", "j", 1, "worker number, concurrent mode if larger than 1")
	cc.cmd.Flags().StringP("repo-type", "", "", "repository type of dest registry server (can be 'harbor' or empty string)")
	cc.cmd.Flags().StringP("default-project", "", "library", "project name (also called 'namespace') when destination image project is empty")
	cc.cmd.Flags().BoolP("harbor-https", "", true, "use https when create harbor project")

	return cc
}

func (cc *mirrorCmd) setupFlags() error {
	configData := config.DefaultProvider.Get("")
	b, _ := json.MarshalIndent(configData, "", "  ")
	logrus.Debugf("config: %v", string(b))
	// command line parameter is prior than env variable
	if config.GetString("source") == "" && utils.EnvSourceRegistry != "" {
		config.Set("source", utils.EnvSourceRegistry)
	}
	if config.GetString("destination") == "" && utils.EnvDestRegistry != "" {
		config.Set("destination", utils.EnvDestRegistry)
	}

	return nil
}

func (cc *mirrorCmd) processImageList() error {
	logrus.Debugf("source registry %q", config.GetString("source"))
	logrus.Debugf("destination registry %q", config.GetString("destination"))
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
		spec := newMirrorImageListSpec(l)
		if spec == nil {
			logrus.Warnf("ignore line %q in list file: invalid format", l)
			continue
		}
		spec.source = utils.ConstructRegistry(
			spec.source, config.GetString("source"))
		spec.destination = utils.ConstructRegistry(
			spec.destination, config.GetString("destination"))
		// only get the destination registry and login
		reg := utils.GetRegistryName(spec.destination)
		if _, ok := cc.registriesSet[reg]; !ok {
			cc.registriesSet[reg] = struct{}{}
		}
		cc.listSpec = append(cc.listSpec, spec)
		if utils.GetProjectName(spec.destination) == "" {
			logrus.Infof("project name of %q is empty, set to %q",
				spec.destination,
				config.GetString("default-project"))
			spec.destination = utils.ReplaceProjectName(
				spec.destination, config.GetString("default-project"))
		}
	}

	for r := range cc.registriesSet {
		if err := cc.baseCmd.runDockerLogin(r); err != nil {
			// output login failed message only
			logrus.Warn(err)
		}
	}

	return nil
}

func (cc *mirrorCmd) createHarborProject() {
	// create harbor project if not exist
	repoType := config.GetString("repo-type")
	if repoType != "harbor" {
		return
	}
	logrus.Infof("start creating harbor projects...")
	dstProjMap := map[string]bool{}
	for _, v := range cc.listSpec {
		dstReg := utils.GetRegistryName(v.destination)
		dstProj := utils.GetProjectName(v.destination)
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
				logrus.Errorf("failed to create harbor project %q: %q",
					dstProj, err)
			}
			dstProjMap[dstProj] = true
		}
	}
}

func (cc *mirrorCmd) run() {
	for i, v := range cc.listSpec {
		m := mirror.NewMirror(&mirror.MirrorOptions{
			Source:      v.source,
			Destination: v.destination,
			Tag:         v.tag,
			ArchList:    strings.Split(config.GetString("arch"), ","),
			Line:        v.line,
			Mode:        mirror.MODE_MIRROR,
			ID:          i + 1,
		})
		cc.baseCmd.workerChan <- m
	}
}

type mirrorImageListSpec struct {
	source      string
	destination string
	tag         string
	line        string
}

func newMirrorImageListSpec(l string) *mirrorImageListSpec {
	v := make([]string, 0, 3)
	s := strings.Split(l, " ")
	for i := range s {
		if s[i] != "" {
			v = append(v, s[i])
		}
	}
	if len(v) != 3 {
		return nil // invalid line format
	}
	return &mirrorImageListSpec{
		source:      v[0],
		destination: v[1],
		tag:         v[2],
		line:        l,
	}
}
