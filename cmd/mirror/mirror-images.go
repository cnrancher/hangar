package mirrorer

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

// mirror COMMAND reads file from image-list txt or stdin, then mirror images
// from source repo to the destination repo
var (
	cmd          = flag.NewFlagSet("mirror", flag.ExitOnError)
	cmdFile      = cmd.String("f", "", "image list file")
	cmdArch      = cmd.String("a", "amd64,arm64", "architecture list of images, seperate with ','")
	cmdSourceReg = cmd.String("s", "", "override the source registry")
	cmdDestReg   = cmd.String("d", "", "override the destination registry")
	cmdFailed    = cmd.String("o", "mirror-failed.txt", "file name of the mirror failed image list")
	cmdDebug     = cmd.Bool("debug", false, "enable the debug output")
	cmdJobs      = cmd.Int("j", 1, "job number, async mode if larger than 1, maximun is 20")
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

	if *cmdSourceReg != "" {
		logrus.Infof("Set source registry to [%s]", *cmdSourceReg)
	} else {
		logrus.Infof("Set source registry to [%s]", u.DockerHubRegistry)
	}

	// Command line parameter is prior than environment variable
	if *cmdDestReg == "" && u.EnvDockerRegistry != "" {
		*cmdDestReg = u.EnvDockerRegistry
	}

	if *cmdDestReg != "" {
		logrus.Infof("Set destination registry to [%s]", *cmdDestReg)
	} else {
		logrus.Infof("Set destination registry to [%s]", u.DockerHubRegistry)
	}

	// execute docker login command
	err := registry.DockerLogin(
		*cmdDestReg, u.EnvDockerUsername, u.EnvDockerPassword)
	if err != nil {
		logrus.Fatalf("MirrorImages login failed: %v", err.Error())
	}

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
			fmt.Println(err)
		}
		defer readFile.Close()

		scanner = bufio.NewScanner(readFile)
		scanner.Split(bufio.ScanLines)
	}

	u.CheckWorkerNum(usingStdin, cmdJobs)
	if !usingStdin {
		logrus.Infof("Creating %d job workers", *cmdJobs)
	} else {
		fmt.Printf(">>> ")
	}
	u.MirrorerJobNum = *cmdJobs

	u.DeleteIfExist(*cmdFailed)
	var writeFileMutex sync.Mutex
	var wg sync.WaitGroup
	// worker function for goroutine pool
	worker := func(id int, ch chan mirror.Mirrorer) {
		defer wg.Done()
		for m := range ch {
			m.SetID(fmt.Sprintf("%02d", id))

			logrus.WithField("M_ID", m.ID()).
				Infof("SOURCE: [%v] DEST: [%v] TAG: [%v]",
					m.Source(), m.Destination(), m.Tag())

			err := m.StartMirror()
			if err != nil {
				logrus.WithField("M_ID", m.ID()).
					Errorf("Failed to copy image [%s]", m.Source())
				logrus.WithField("M_ID", m.ID()).
					Error("Mirror", err.Error())
				writeFileMutex.Lock()
				u.AppendFileLine(*cmdFailed, fmt.Sprintf("%s %s %s",
					m.Source(), m.Destination(), m.Tag()))
				writeFileMutex.Unlock()
			} else if m.ImageNum()-m.Copied() != 0 {
				// if there are some images copy failed in this mirrorer
				logrus.WithField("M_ID", m.ID()).
					Errorf("Some images failed to mirror: %s", m.Source())
				writeFileMutex.Lock()
				u.AppendFileLine(*cmdFailed, fmt.Sprintf("%s %s %s",
					m.Source(), m.Destination(), m.Tag()))
				writeFileMutex.Unlock()
			}
			if usingStdin {
				fmt.Printf(">>> ")
			}
		}
	}
	mirrorChan := make(chan mirror.Mirrorer)
	for i := 0; i < *cmdJobs; i++ {
		wg.Add(1)
		go worker(i+1, mirrorChan)
	}

	for scanner.Scan() {
		l := scanner.Text()
		// Ignore empty/comment line
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		var v []string = make([]string, 0, 4)
		for _, s := range strings.Split(l, " ") {
			if s != "" {
				v = append(v, s)
			}
		}
		if len(v) != 3 {
			logrus.Errorf("Invalid line format")
			logrus.Errorf("Should be: '<SOURCE> <DESTINATION> <TAG>'")
			logrus.Debugf("Skip line %s", l)
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		var mirrorer mirror.Mirrorer = mirror.NewMirror(&mirror.MirrorOptions{
			Source:      u.ConstructRegistry(v[0], *cmdSourceReg),
			Destination: u.ConstructRegistry(v[1], *cmdDestReg),
			Tag:         v[2],
			ArchList:    strings.Split(*cmdArch, ","),
			Mode:        mirror.MODE_MIRROR,
		})

		mirrorChan <- mirrorer
	}

	close(mirrorChan)
	wg.Wait()
	if usingStdin {
		fmt.Println()
	}
}
