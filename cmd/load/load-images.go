package loader

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cnrancher.io/image-tools/archive"
	command "cnrancher.io/image-tools/cmd"
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

	cmdDefaultProject = cmd.String("default-project", "library", "project name when project is empty")
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
	if *cmdDestReg == "" {
		logrus.Error("dest registry not specified.")
		logrus.Errorf("Use '-d' to specify the dest registry:port")
		logrus.Fatal("Failed to load images.")
	}

	if err := registry.SelfCheckSkopeo(); err != nil {
		logrus.Error("registry self check skopeo failed.")
		logrus.Fatal(err)
	}
	if err := registry.SelfCheckBuildX(); err != nil {
		logrus.Error("registry self check failed.")
		logrus.Fatal(err)
	}

	var compressFormat archive.CompressFormat = archive.CompressFormatGzip
	switch *cmdCompress {
	case "gzip":
		compressFormat = archive.CompressFormatGzip
	case "zstd":
		compressFormat = archive.CompressFormatZstd
	case "dir":
		compressFormat = archive.CompressFormatDirectory
	default:
		compressFormat = archive.CompressFormatGzip
	}

	// Check cache image directory
	if compressFormat != archive.CompressFormatDirectory {
		if err := u.CheckCacheDirEmpty(); err != nil {
			logrus.Fatal(err)
		}
	}

	// Command line parameter is prior than environment variable
	if *cmdDestReg == "" && u.EnvDestRegistry != "" {
		*cmdDestReg = u.EnvDestRegistry
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
		ext := filepath.Ext(*cmdSource)
		if strings.Contains(ext, "part") {
			logrus.Warnf("File name %q contains 'part*' extention", *cmdSource)
			*cmdSource = strings.TrimRight(*cmdSource, ext)
			logrus.Warnf("Rename it to %q", *cmdSource)
		}
		logrus.Infof("Decompressing %s...", *cmdSource)
		err := archive.Decompress(*cmdSource, directory, compressFormat)
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

	u.DeleteIfExist(*cmdFailed)
	u.CheckWorkerNum(false, cmdJobs)
	logrus.Infof("Creating %d job workers", *cmdJobs)
	u.WorkerNum = *cmdJobs
	ch, wg := command.Worker(*cmdJobs, *cmdFailed, nil)
	if err := command.DockerLoginRegistry(*cmdDestReg); err != nil {
		logrus.Error(err)
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
			user, passwd, _ := registry.GetDockerPassword(*cmdDestReg)
			err := registry.CreateHarborProject(proj, url, user, passwd)
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
		ch <- m
	}

	close(ch)
	wg.Wait()
}
