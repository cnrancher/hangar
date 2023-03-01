package load

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	command "github.com/cnrancher/hangar/cmd"
	"github.com/cnrancher/hangar/pkg/archive"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/cnrancher/hangar/pkg/registry"
	u "github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	cmdSource         string
	cmdDestReg        string
	cmdFailed         string
	cmdRepoType       string
	cmdCompress       string
	cmdHarborHttps    bool
	cmdDebug          bool
	cmdJobs           int
	cmdDefaultProject string
	flagSet           = flag.NewFlagSet("load", flag.ExitOnError)
)

func Parse(args []string) {
	flagSet.StringVar(&cmdSource, "s", "",
		"saved file to load (can use '-compress' to specify the input file format, default is 'gzip')")
	flagSet.StringVar(&cmdDestReg, "d", "", "target private registry:port")
	flagSet.StringVar(&cmdFailed, "o", "load-failed.txt", "file name of the load failed image list")
	flagSet.StringVar(&cmdRepoType, "repo-type", "", "repository type, can be 'harbor' or empty")
	flagSet.StringVar(&cmdCompress, "compress", "gzip", "compress format, can be 'gzip', 'zstd' or 'dir'")
	flagSet.StringVar(&cmdDefaultProject, "default-project", "library", "project name when project is empty")
	flagSet.BoolVar(&cmdHarborHttps, "harbor-https", true, "use HTTPS by default when create harbor project")
	flagSet.BoolVar(&cmdDebug, "debug", false, "enable the debug output")
	flagSet.IntVar(&cmdJobs, "j", 1, "job number, async mode if larger than 1, maximum is 20")

	flagSet.Parse(args)
}

func LoadImages() {
	var err error
	if cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if cmdSource == "" {
		logrus.Error("saved file not specified.")
		logrus.Error("Use '-s' to specify the file name (tarball or directory)")
		logrus.Fatal("Failed to load images.")
	}
	if cmdDestReg == "" {
		logrus.Error("dest registry not specified.")
		logrus.Errorf("Use '-d' to specify the dest registry:port")
		logrus.Fatal("Failed to load images.")
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

	var compressFormat archive.CompressFormat = archive.CompressFormatGzip
	switch cmdCompress {
	case "gzip":
		compressFormat = archive.CompressFormatGzip
	case "zstd":
		compressFormat = archive.CompressFormatZstd
	case "dir":
		compressFormat = archive.CompressFormatDirectory
	default:
		logrus.Warnf("Unknow compress format %q, set to gzip", cmdCompress)
		compressFormat = archive.CompressFormatGzip
	}

	// Check cache image directory
	if compressFormat != archive.CompressFormatDirectory {
		if err := u.CheckCacheDirEmpty(); err != nil {
			logrus.Fatal(err)
		}
	}

	// Command line parameter is prior than environment variable
	if cmdDestReg == "" && u.EnvDestRegistry != "" {
		cmdDestReg = u.EnvDestRegistry
	}
	if err := command.ProcessDockerLoginEnv(); err != nil {
		logrus.Error(err)
	}

	directory := "."
	if directory, err = u.GetAbsPath(directory); err != nil {
		logrus.Fatal(err)
	}

	if compressFormat != archive.CompressFormatDirectory {
		// decompress input tar.* tarball
		// if filename already have '.part*' extention
		ext := filepath.Ext(cmdSource)
		if strings.Contains(ext, "part") {
			logrus.Warnf("File name %q contains 'part*' extention", cmdSource)
			cmdSource = strings.TrimRight(cmdSource, ext)
			logrus.Warnf("Rename it to %q", cmdSource)
		}
		logrus.Infof("Decompressing %s...", cmdSource)
		err := archive.Decompress(cmdSource, directory, compressFormat)
		if err != nil {
			logrus.Fatal(err)
		}
		directory = filepath.Join(directory, u.CacheImageDirectory)
		logrus.Debugf("Decompressed directory: %s", directory)
	} else {
		directory = filepath.Join(directory, cmdSource)
	}
	info, err := os.Stat(directory)
	if err != nil {
		logrus.Fatal(err.Error())
	}
	if !info.IsDir() {
		logrus.Fatalf("'%s' is not a directory", directory)
	}

	u.DeleteIfExist(cmdFailed)
	u.CheckWorkerNum(false, &cmdJobs)
	logrus.Infof("Creating %d job workers", cmdJobs)
	u.WorkerNum = cmdJobs
	ch, wg := command.Worker(cmdFailed, nil)
	if err := command.DockerLoginRegistry(cmdDestReg); err != nil {
		logrus.Error(err)
	}

	var mList []*mirror.Mirror
	if cmdRepoType == "harbor" {
		// Set default project name if dest repo is harbor
		mList, err = mirror.LoadSavedTemplates(
			directory, cmdDestReg, cmdDefaultProject)
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
			url := fmt.Sprintf("%s/api/v2.0/projects", cmdDestReg)
			if cmdHarborHttps {
				url = "https://" + url
			} else {
				url = "http://" + url
			}
			user, passwd, _ := registry.GetDockerPassword(cmdDestReg)
			err := registry.CreateHarborProject(proj, url, user, passwd)
			if err != nil {
				logrus.Errorf("Failed to create harbor project %q: %q",
					proj, err)
			}
		}
	} else {
		// Do not set default project name if dest repo is not harbor
		mList, err = mirror.LoadSavedTemplates(directory, cmdDestReg, "")
		if err != nil {
			logrus.Fatal(err)
		}
	}

	for _, m := range mList {
		ch <- m
	}

	close(ch)
	wg.Wait()
}
