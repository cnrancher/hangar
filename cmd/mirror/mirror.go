package mirror

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	command "github.com/cnrancher/hangar/cmd"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/cnrancher/hangar/pkg/registry"
	u "github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	cmdFile           string
	cmdArch           string
	cmdSourceReg      string
	cmdDestReg        string
	cmdFailed         string
	cmdDebug          bool
	cmdJobs           int
	cmdRepoType       string
	cmdHarborHttps    bool
	cmdDefaultProject string
	flagSet           = flag.NewFlagSet("mirror", flag.ExitOnError)
)

func Parse(args []string) {
	flagSet.StringVar(&cmdFile, "f", "", "image list file")
	flagSet.StringVar(&cmdArch, "a", "amd64,arm64", "architecture list of images, separate with ','")
	flagSet.StringVar(&cmdSourceReg, "s", "", "override the source registry")
	flagSet.StringVar(&cmdDestReg, "d", "", "override the destination registry")
	flagSet.StringVar(&cmdFailed, "o", "mirror-failed.txt", "file name of the mirror failed image list")
	flagSet.BoolVar(&cmdDebug, "debug", false, "enable the debug output")
	flagSet.IntVar(&cmdJobs, "j", 1, "job number, async mode if larger than 1, maximun is 20")
	flagSet.StringVar(&cmdRepoType, "repo-type", "", "repository type, can be 'harbor' or empty")
	flagSet.BoolVar(&cmdHarborHttps, "harbor-https", true, "use HTTPS by default when create harbor project")
	flagSet.StringVar(&cmdDefaultProject, "default-project", "library", "project name when dest project is empty")
	flagSet.Parse(args)
}

func MirrorImages() {
	if cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	var selfCheckFailed = false
	if err := registry.SelfCheckSkopeo(); err != nil {
		logrus.Error("self check skopeo failed.")
		logrus.Error(err)
		selfCheckFailed = true
	}
	if err := registry.SelfCheckBuildX(); err != nil {
		logrus.Error("self check docker-buildx failed.")
		logrus.Error(err)
		selfCheckFailed = true
	}
	if selfCheckFailed {
		os.Exit(1)
	}

	// Command line parameter is prior than environment variable
	if cmdDestReg == "" && u.EnvDestRegistry != "" {
		cmdDestReg = u.EnvDestRegistry
	}
	if cmdSourceReg == "" && u.EnvSourceRegistry != "" {
		cmdSourceReg = u.EnvSourceRegistry
	}
	logrus.Debugf("Source registry %q", cmdSourceReg)
	logrus.Debugf("Dest registry %q", cmdDestReg)
	if err := command.ProcessDockerLoginEnv(); err != nil {
		logrus.Warn(err)
	}

	var registryMap = make(map[string]bool)
	var scanner *bufio.Scanner
	var usingStdin bool
	if cmdFile == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE> <DESTINATION> <TAG>' from stdin")
		logrus.Info("Use 'Ctrl+C' or 'Ctrl+D' to exit.")
	} else {
		readFile, err := os.Open(cmdFile)
		if err != nil {
			logrus.Fatal(err)
		}
		defer readFile.Close()
		// pre-load dest registries in image list
		sc := bufio.NewScanner(readFile)
		sc.Split(bufio.ScanLines)
		for sc.Scan() {
			v := processImageListLine(sc.Text())
			if len(v) != 3 {
				continue
			}
			// add dest registry into registry map
			reg := u.GetRegistryName(u.ConstructRegistry(v[1], cmdDestReg))
			if !registryMap[reg] {
				registryMap[reg] = true
			}
		}

		// Run docker login for all registries in image list before mirror
		for k := range registryMap {
			if err := command.DockerLoginRegistry(k); err != nil {
				logrus.Warn(err)
			}
		}

		// reset file seek
		readFile.Seek(0, io.SeekStart)
		scanner = bufio.NewScanner(readFile)
		scanner.Split(bufio.ScanLines)
	}

	u.CheckWorkerNum(usingStdin, &cmdJobs)
	if !usingStdin {
		logrus.Infof("Creating %d job workers", cmdJobs)
	} else {
		fmt.Printf(">>> ")
	}
	u.WorkerNum = cmdJobs
	u.DeleteIfExist(cmdFailed)
	ch, wg := command.Worker(cmdFailed, func(m *mirror.Mirror) {
		if usingStdin {
			fmt.Printf(">>> ")
		}
	})
	var destProjects map[string]bool = make(map[string]bool)
	var num int = 0
	for scanner.Scan() {
		l := scanner.Text()
		v := processImageListLine(l)
		if len(v) != 3 {
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		num++
		destReg := u.GetRegistryName(u.ConstructRegistry(v[1], cmdDestReg))
		registryMap := make(map[string]bool)
		if usingStdin && !registryMap[destReg] {
			if err := command.DockerLoginRegistry(destReg); err != nil {
				logrus.Error(err)
			} else {
				registryMap[destReg] = true
			}
		}
		m := mirror.NewMirror(&mirror.MirrorOptions{
			Source:      u.ConstructRegistry(v[0], cmdSourceReg),
			Destination: u.ConstructRegistry(v[1], cmdDestReg),
			Tag:         v[2],
			ArchList:    strings.Split(cmdArch, ","),
			Line:        l,
			Mode:        mirror.MODE_MIRROR,
			ID:          num,
		})

		// If the dest image project name is empty
		if u.GetProjectName(m.Destination) == "" {
			logrus.Warnf("The project of %q is empty, set to default %q",
				m.Destination, cmdDefaultProject)
			m.Destination = u.ReplaceProjectName(
				m.Destination, cmdDefaultProject)
		}
		if cmdRepoType == "harbor" {
			destReg := u.GetRegistryName(m.Destination)
			destProj := u.GetProjectName(m.Destination)
			// Create the project name
			if !destProjects[destProj] {
				url := fmt.Sprintf("%s/api/v2.0/projects", destReg)
				if cmdHarborHttps {
					url = "https://" + url
				} else {
					url = "http://" + url
				}
				user, passwd, _ := registry.GetDockerPassword(destReg)
				err := registry.CreateHarborProject(destProj, url, user, passwd)
				if err != nil {
					logrus.Errorf("Failed to create harbor project %q: %q",
						destProj, err)
				}
				destProjects[destProj] = true
			}
		}
		ch <- m
	}

	close(ch)
	wg.Wait()
	if usingStdin {
		fmt.Println()
	}
}

func processImageListLine(l string) []string {
	var spec []string = make([]string, 0)
	if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
		return spec
	}
	var v []string = make([]string, 0, 4)
	for _, s := range strings.Split(l, " ") {
		if s != "" {
			v = append(v, s)
		}
	}
	if len(v) != 3 {
		return spec
	}
	return v
}
