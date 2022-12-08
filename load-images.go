package main

import (
	"fmt"
	"os"
	"sync"

	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func LoadImages() {
	if dockerUsername == "" || dockerPassword == "" {
		logrus.Fatal("DOCKER_USERNAME, DOCKER_PASSWORD environment not set")
		// TODO: read username and password from stdin
	}

	if err := registry.SelfCheck(); err != nil {
		logrus.Error("registry self check failed.")
		logrus.Fatal(err)
	}

	// Command line parameter is prior than environment variable
	if *loadDestReg == "" && dockerRegistry != "" {
		*loadDestReg = dockerRegistry
	}

	if *loadDestReg != "" {
		logrus.Infof("Set destination registry to [%s]", *loadDestReg)
	} else {
		logrus.Infof("Set destination registry to [%s]", u.DockerHubRegistry)
	}

	// execute docker login command
	err := registry.DockerLogin(*loadDestReg, dockerUsername, dockerPassword)
	if err != nil {
		logrus.Fatalf("MirrorImages login failed: %v", err.Error())
	}

	// TODO: decompress tar.gz tarball
	// TODO:
	directory := *loadFile
	if directory == "" {
		directory = "output/"
	}

	if directory, err = u.GetAbsPath(directory); err != nil {
		logrus.Fatal(err)
	}
	logrus.Debugf("Decompressed directory: %s", directory)
	info, err := os.Stat(directory)
	if err != nil {
		logrus.Fatal(err.Error())
	}
	if !info.IsDir() {
		logrus.Fatalf("'%s' is not a directory", directory)
	}

	if *loadJobs > 20 {
		logrus.Warn("worker count should be <= 20")
		logrus.Warn("change worker count to 20")
		*loadJobs = 20
	} else if *loadJobs < 1 {
		logrus.Warn("invalid worker count")
		logrus.Warn("change worker count to 1")
		*loadJobs = 20
	}
	logrus.Infof("Creating %d job workers", *loadJobs)
	u.MirrorerJobNum = *loadJobs

	var writeFileMutex sync.Mutex
	var wg sync.WaitGroup
	// worker function for goroutine pool
	worker := func(id int, ch chan mirror.Mirrorer) {
		defer wg.Done()
		for m := range ch {
			m.SetID(fmt.Sprintf("%02d", id))

			logrus.WithField("M_ID", m.ID()).
				Infof("DEST: [%v] TAG: [%v]", m.Destination(), m.Tag())

			err := m.StartLoad()
			if err != nil {
				logrus.WithField("M_ID", m.ID()).
					Errorf("Failed to load image [%s]", m.Destination())
				logrus.WithField("M_ID", m.ID()).
					Error("Mirror", err.Error())
				writeFileMutex.Lock()
				u.AppendFileLine(*loadFailed,
					fmt.Sprintf("%s:%s\n", m.Destination(), m.Tag()))
				writeFileMutex.Unlock()
			} else if m.ImageNum()-m.Loaded() != 0 {
				// if there are some images load failed in this mirrorer
				logrus.WithField("M_ID", m.ID()).
					Errorf("Some images failed to load: %s", m.Source())
				writeFileMutex.Lock()
				u.AppendFileLine(*loadFailed,
					fmt.Sprintf("%s:%s\n", m.Destination(), m.Tag()))
				writeFileMutex.Unlock()
			}
		}
	}
	mirrorerChan := make(chan mirror.Mirrorer)
	for i := 0; i < *mirrorJobs; i++ {
		wg.Add(1)
		go worker(i+1, mirrorerChan)
	}

	mirrorerList, err := mirror.LoadSavedTemplates(directory, *loadDestReg)
	if err != nil {
		logrus.Fatal(err)
	}
	for _, m := range mirrorerList {
		mirrorerChan <- m
	}

	close(mirrorerChan)
	wg.Wait()
}
