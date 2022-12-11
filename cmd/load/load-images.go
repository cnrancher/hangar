package loader

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

var (
	cmd        = flag.NewFlagSet("load", flag.ExitOnError)
	cmdFile    = cmd.String("f", "", "saved tar.gz file")
	cmdDestReg = cmd.String("d", "", "override the destination registry")
	cmdFailed  = cmd.String("o", "load-failed.txt", "file name of the load failed image list")
	cmdDebug   = cmd.Bool("debug", false, "enable the debug output")
	cmdJobs    = cmd.Int("j", 1, "job number, async mode if larger than 1, maximum is 20")
)

func Parse(args []string) {
	cmd.Parse(args)
}

func LoadImages() {
	if *cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := registry.SelfCheckBuildX(); err != nil {
		logrus.Error("registry self check failed.")
		logrus.Fatal(err)
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

	// TODO: decompress tar.gz tarball
	// TODO:
	directory := *cmdFile
	if directory == "" {
		directory = u.CacheImageDirectory
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

	u.CheckWorkerNum(false, cmdJobs)
	logrus.Infof("Creating %d job workers", *cmdJobs)
	u.WorkerNum = *cmdJobs

	u.DeleteIfExist(*cmdFailed)
	var writeFileMutex sync.Mutex
	var wg sync.WaitGroup
	// worker function for goroutine pool
	worker := func(id int, ch chan *mirror.Mirror) {
		defer wg.Done()
		for m := range ch {
			err := m.StartLoad()
			if err != nil {
				logrus.WithField("M_ID", m.MID).
					Errorf("Failed to load image [%s]", m.Destination)
				logrus.WithField("M_ID", m.MID).
					Error("Mirror", err.Error())
				writeFileMutex.Lock()
				u.AppendFileLine(*cmdFailed,
					fmt.Sprintf("%s:%s\n", m.Destination, m.Tag))
				writeFileMutex.Unlock()
			} else if m.ImageNum()-m.Loaded() != 0 {
				// if there are some images load failed in this mirrorer
				logrus.WithField("M_ID", m.MID).
					Errorf("Some images failed to load: %s", m.Source)
				writeFileMutex.Lock()
				u.AppendFileLine(*cmdFailed,
					fmt.Sprintf("%s:%s\n", m.Destination, m.Tag))
				writeFileMutex.Unlock()
			}
		}
	}
	mChan := make(chan *mirror.Mirror)
	for i := 0; i < *cmdJobs; i++ {
		wg.Add(1)
		go worker(i+1, mChan)
	}

	mList, err := mirror.LoadSavedTemplates(directory, *cmdDestReg)
	if err != nil {
		logrus.Fatal(err)
	}
	for _, m := range mList {
		mChan <- m
	}

	close(mChan)
	wg.Wait()
}
