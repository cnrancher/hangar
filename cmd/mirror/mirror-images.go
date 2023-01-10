package mirrorer

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	command "cnrancher.io/image-tools/cmd"
	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

// mirror COMMAND reads file from image-list txt or stdin, then mirror images
// from source repo to the destination repo
var (
	cmd            = flag.NewFlagSet("mirror", flag.ExitOnError)
	cmdFile        = cmd.String("f", "", "image list file")
	cmdArch        = cmd.String("a", "amd64,arm64", "architecture list of images, separate with ','")
	cmdSourceReg   = cmd.String("s", "", "override the source registry")
	cmdDestReg     = cmd.String("d", "", "override the destination registry")
	cmdFailed      = cmd.String("o", "mirror-failed.txt", "file name of the mirror failed image list")
	cmdDebug       = cmd.Bool("debug", false, "enable the debug output")
	cmdJobs        = cmd.Int("j", 1, "job number, async mode if larger than 1, maximun is 20")
	cmdRepoType    = cmd.String("repo-type", "", "repository type, can be 'harbor' or empty")
	cmdHarborHttps = cmd.Bool("harbor-https", true, "use HTTPS by default when create harbor project")

	cmdDefaultProject = cmd.String("default-project", "library", "project name when dest project is empty")
)

func Parse(args []string) {
	cmd.Parse(args)
}

func MirrorImages() {
	if *cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if err := registry.SelfCheckSkopeo(); err != nil {
		logrus.Error("registry self check skopeo failed.")
		logrus.Fatal(err)
	} else if err = registry.SelfCheckBuildX(); err != nil {
		logrus.Error("registry self check buildx failed.")
		logrus.Fatal(err)
	}

	// Command line parameter is prior than environment variable
	if *cmdDestReg == "" && u.EnvDestRegistry != "" {
		*cmdDestReg = u.EnvDestRegistry
	}
	if *cmdSourceReg == "" && u.EnvSourceRegistry != "" {
		*cmdSourceReg = u.EnvSourceRegistry
	}
	logrus.Debugf("Source registry %q", *cmdSourceReg)
	logrus.Debugf("Dest registry %q", *cmdDestReg)
	if err := command.ProcessDockerLoginEnv(); err != nil {
		logrus.Warn(err)
	}

	var registryMap = make(map[string]bool)
	var scanner *bufio.Scanner
	var usingStdin bool
	if *cmdFile == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE> <DESTINATION> <TAG>' from stdin")
		logrus.Info("Use 'Ctrl+C' or 'Ctrl+D' to exit.")
	} else {
		readFile, err := os.Open(*cmdFile)
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
			reg := u.GetRegistryName(u.ConstructRegistry(v[1], *cmdDestReg))
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

	u.CheckWorkerNum(usingStdin, cmdJobs)
	if !usingStdin {
		logrus.Infof("Creating %d job workers", *cmdJobs)
	} else {
		fmt.Printf(">>> ")
	}
	u.WorkerNum = *cmdJobs
	u.DeleteIfExist(*cmdFailed)
	ch, wg := command.Worker(*cmdJobs, *cmdFailed, func(m *mirror.Mirror) {
		if usingStdin {
			fmt.Println(">>> ")
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
		destReg := u.GetRegistryName(u.ConstructRegistry(v[1], *cmdDestReg))
		registryMap := make(map[string]bool)
		if usingStdin && !registryMap[destReg] {
			if err := command.DockerLoginRegistry(destReg); err != nil {
				logrus.Error(err)
			} else {
				registryMap[destReg] = true
			}
		}
		m := mirror.NewMirror(&mirror.MirrorOptions{
			Source:      u.ConstructRegistry(v[0], *cmdSourceReg),
			Destination: u.ConstructRegistry(v[1], *cmdDestReg),
			Tag:         v[2],
			ArchList:    strings.Split(*cmdArch, ","),
			Line:        l,
			Mode:        mirror.MODE_MIRROR,
			ID:          num,
		})

		// If the dest image project name is empty
		if u.GetProjectName(m.Destination) == "" {
			logrus.Warnf("The project of %q is empty, set to default %q",
				m.Destination, *cmdDefaultProject)
			m.Destination = u.ReplaceProjectName(
				m.Destination, *cmdDefaultProject)
		}
		if *cmdRepoType == "harbor" {
			destReg := u.GetRegistryName(m.Destination)
			destProj := u.GetProjectName(m.Destination)
			// Create the project name
			if !destProjects[destProj] {
				url := fmt.Sprintf("%s/api/v2.0/projects", destReg)
				if *cmdHarborHttps {
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
