package loader

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

var (
	cmd            = flag.NewFlagSet("load", flag.ExitOnError)
	cmdSource      = cmd.String("s", "", "saved file to load (tar tarball or a directory)")
	cmdDestReg     = cmd.String("d", "", "target private registry:port")
	cmdFailed      = cmd.String("o", "load-failed.txt", "file name of the load failed image list")
	cmdRepoType    = cmd.String("repo-type", "", "repository type, can be 'harbor' or empty")
	cmdCompress    = cmd.String("compress", "gzip", "compress format, can be 'gzip', 'zstd' or 'dir'")
	cmdHarborHttps = cmd.Bool("harbor-https", true, "use HTTPS by default when create harbor project")
	cmdDebug       = cmd.Bool("debug", false, "enable the debug output")
	cmdJobs        = cmd.Int("j", 1, "job number, async mode if larger than 1, maximum is 20")

	cmdDefaultProject = cmd.String("default-project", "library", "project name when dest repo type is harbor and dest project is empty")
)

func Parse(args []string) {
	cmd.Parse(args)
}

func LoadImages() {
	var err error
	if *cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *cmdSource == "" {
		logrus.Error("saved file not specified.")
		logrus.Error("Use '-s' to specify the file name (tarball or directory)")
		logrus.Fatal("Failed to load images.")
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
		logrus.Infof("Set 'docker login' registry to %q", *cmdDestReg)
	} else {
		logrus.Infof("Set 'docker login' registry to %q", u.DockerHubRegistry)
	}

	var compressFormat u.CompressFormat = u.CompressFormatGzip
	switch *cmdCompress {
	case "gzip":
		compressFormat = u.CompressFormatGzip
	case "zstd":
		compressFormat = u.CompressFormatZstd
	case "dir":
		compressFormat = u.CompressFormatDirectory
	default:
		compressFormat = u.CompressFormatGzip
	}

	// Check cache image directory
	if compressFormat != u.CompressFormatDirectory {
		if err := u.CheckCacheDirEmpty(); err != nil {
			logrus.Fatal(err)
		}
	}

	// execute docker login command
	if err := registry.DockerLogin(*cmdDestReg); err != nil {
		logrus.Fatalf("MirrorImages login failed: %v", err.Error())
	}

	directory := "."
	if directory, err = u.GetAbsPath(directory); err != nil {
		logrus.Fatal(err)
	}

	if compressFormat != u.CompressFormatDirectory {
		// decompress input tar.gz tarball
		logrus.Infof("Decompressing %s...", *cmdSource)
		err := u.Decompress(*cmdSource, directory, compressFormat)
		if err != nil {
			logrus.Fatal(err)
		}
		directory = filepath.Join(directory, u.CacheImageDirectory)
		logrus.Debugf("Decompressed directory: %s", directory)
	} else {
		directory = filepath.Join(directory, *cmdSource)
	}
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
					fmt.Sprintf("%s:%s", m.Destination, m.Tag))
				writeFileMutex.Unlock()
			} else if m.ImageNum()-m.Loaded() != 0 {
				// if there are some images load failed in this mirrorer
				logrus.WithField("M_ID", m.MID).
					Errorf("Some images failed to load: %s", m.Source)
				writeFileMutex.Lock()
				u.AppendFileLine(*cmdFailed,
					fmt.Sprintf("%s:%s", m.Destination, m.Tag))
				writeFileMutex.Unlock()
			}
		}
	}
	mChan := make(chan *mirror.Mirror)
	for i := 0; i < *cmdJobs; i++ {
		wg.Add(1)
		go worker(i+1, mChan)
	}

	var mList []*mirror.Mirror
	if *cmdRepoType == "harbor" {
		// Set default project name if dest repo is harbor
		mList, err = mirror.LoadSavedTemplates(
			directory, *cmdDestReg, *cmdDefaultProject)
		if err != nil {
			logrus.Fatal(err)
		}
		var projMap = make(map[string]bool, 0)
		for _, m := range mList {
			// create harbor project before load
			proj := u.GetProjectName(m.Destination)
			if projMap[proj] {
				continue
			}
			projMap[proj] = true
			url := fmt.Sprintf("%s/api/v2.0/projects", *cmdDestReg)
			if *cmdHarborHttps {
				url = "https://" + url
			} else {
				url = "http://" + url
			}
			err := registry.CreateHarborProject(proj, url)
			if err != nil {
				logrus.Errorf("Failed to create harbor project %q: %q",
					proj, err)
			}
		}
	} else {
		// Do not set default project name if dest repo is not harbor
		mList, err = mirror.LoadSavedTemplates(directory, *cmdDestReg, "")
		if err != nil {
			logrus.Fatal(err)
		}
	}

	for _, m := range mList {
		mChan <- m
	}

	close(mChan)
	wg.Wait()
}
