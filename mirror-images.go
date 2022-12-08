package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func MirrorImages() {
	if err := registry.SelfCheck(); err != nil {
		logrus.Error("registry self check failed.")
		logrus.Fatal(err)
	}

	if dockerUsername == "" || dockerPassword == "" {
		logrus.Fatal("DOCKER_USERNAME, DOCKER_PASSWORD environment not set")
		// TODO: read username and password from stdin
	}

	if *mirrorSourceReg != "" {
		logrus.Infof("Set source registry to [%s]", *mirrorSourceReg)
	} else {
		logrus.Infof("Set source registry to [%s]", u.DockerHubRegistry)
	}

	// Command line parameter is prior than environment variable
	if *mirrorDestReg == "" && dockerRegistry != "" {
		*mirrorDestReg = dockerRegistry
	}

	if *mirrorDestReg != "" {
		logrus.Infof("Set destination registry to [%s]", *mirrorDestReg)
	} else {
		logrus.Infof("Set destination registry to [%s]", u.DockerHubRegistry)
	}

	// execute docker login command
	err := registry.DockerLogin(*mirrorDestReg, dockerUsername, dockerPassword)
	if err != nil {
		logrus.Fatalf("MirrorImages login failed: %v", err.Error())
	}

	var scanner *bufio.Scanner
	var usingStdin bool
	if *mirrorFile == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE> <DESTINATION> <TAG>' from stdin")
		logrus.Info("Use 'Ctrl+C' or 'Ctrl+D' to exit.")
	} else {
		readFile, err := os.Open(*mirrorFile)
		if err != nil {
			fmt.Println(err)
		}
		defer readFile.Close()

		scanner = bufio.NewScanner(readFile)
		scanner.Split(bufio.ScanLines)
	}

	if usingStdin && *mirrorJobs != 1 {
		logrus.Warn("async mode not supported in stdin mode")
		logrus.Warn("change worker count back to 1")
		*mirrorJobs = 1
	}
	if *mirrorJobs > 20 {
		logrus.Warn("worker count should be <= 20")
		logrus.Warn("change worker count to 20")
		*mirrorJobs = 20
	}
	if *mirrorJobs < 1 {
		logrus.Warn("invalid worker count")
		logrus.Warn("change worker count to 1")
		*mirrorJobs = 20
	}
	if !usingStdin {
		logrus.Infof("Creating %d job workers", *mirrorJobs)
	} else {
		fmt.Printf(">>> ")
	}
	u.MirrorerJobNum = *mirrorJobs

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
				u.AppendFileLine(*mirrorFailed, fmt.Sprintf("%s %s %s",
					m.Source(), m.Destination(), m.Tag()))
				writeFileMutex.Unlock()
			} else if m.ImageNum()-m.Copied() != 0 {
				// if there are some images copy failed in this mirrorer
				logrus.WithField("M_ID", m.ID()).
					Errorf("Some images failed to mirror: %s", m.Source())
				writeFileMutex.Lock()
				u.AppendFileLine(*mirrorFailed, fmt.Sprintf("%s %s %s",
					m.Source(), m.Destination(), m.Tag()))
				writeFileMutex.Unlock()
			}
			if usingStdin {
				fmt.Printf(">>> ")
			}
		}
	}
	mirrorChan := make(chan mirror.Mirrorer)
	for i := 0; i < *mirrorJobs; i++ {
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
			Source:      mirror.ConstructRegistry(v[0], *mirrorSourceReg),
			Destination: mirror.ConstructRegistry(v[1], *mirrorDestReg),
			Tag:         v[2],
			ArchList:    strings.Split(*mirrorArch, ","),
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
