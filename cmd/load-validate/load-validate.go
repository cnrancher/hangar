package loadvalidate

import (
	"flag"
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
	cmd         = flag.NewFlagSet("load-validate", flag.ExitOnError)
	cmdSource   = cmd.String("s", "", "saved file to load (tar tarball or a directory)")
	cmdDestReg  = cmd.String("d", "", "target private registry:port")
	cmdFailed   = cmd.String("o", "load-validate-failed.txt", "file name of the validate failed image list")
	cmdCompress = cmd.String("compress", "gzip", "compress format, can be 'gzip', 'zstd' or 'dir'")
	cmdDebug    = cmd.Bool("debug", false, "enable the debug output")
	cmdJobs     = cmd.Int("j", 1, "job number, async mode if larger than 1, maximun is 20")

	cmdDefaultProject = cmd.String("default-project", "library", "project name when dest repo type is harbor and dest project is empty")
)

func Parse(args []string) {
	cmd.Parse(args)
}

func LoadValidate() {
	if *cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *cmdSource == "" {
		logrus.Error("saved file not specified.")
		logrus.Error("Use '-s' to specify the file name (tarball or directory)")
		logrus.Fatal("Failed to validate images.")
	}
	if *cmdDestReg == "" {
		logrus.Error("dest registry not specified.")
		logrus.Errorf("Use '-d' to specify the dest registry:port")
		logrus.Fatal("Failed to validate images.")
	}
	if err := registry.SelfCheckSkopeo(); err != nil {
		logrus.Error("registry self check skopeo failed.")
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

	var err error
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
	u.CheckWorkerNum(false, cmdJobs)
	logrus.Infof("Creating %d job workers", *cmdJobs)
	u.WorkerNum = *cmdJobs
	u.DeleteIfExist(*cmdFailed)
	ch, wg := command.Worker(*cmdJobs, *cmdFailed, nil)
	if err := command.DockerLoginRegistry(*cmdDestReg); err != nil {
		logrus.Error(err)
	}
	mList, err := mirror.LoadSavedTemplates(directory, *cmdDestReg, "")
	if err != nil {
		logrus.Fatal(err)
	}
	for i := range mList {
		mList[i].Mode = mirror.MODE_LOAD_VALIDATE
		if u.GetProjectName(mList[i].Source) == "" {
			mList[i].Source = u.ReplaceProjectName(
				mList[i].Source, *cmdDefaultProject)
		}
		if u.GetProjectName(mList[i].Destination) == "" {
			mList[i].Destination = u.ReplaceProjectName(
				mList[i].Destination, *cmdDefaultProject)
		}
		ch <- mList[i]
	}
	close(ch)
	wg.Wait()
}
