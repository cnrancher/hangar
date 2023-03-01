package save

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	command "github.com/cnrancher/hangar/cmd"
	"github.com/cnrancher/hangar/pkg/archive"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/cnrancher/hangar/pkg/registry"
	u "github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	flagSet     = flag.NewFlagSet("save", flag.ExitOnError)
	cmdFile     string
	cmdArch     string
	cmdSource   string
	cmdDest     string
	cmdFailed   string
	cmdCompress string
	cmdPart     bool
	cmdPartSize string
	cmdDebug    bool
	cmdJobs     int
)

func Parse(args []string) {
	flagSet.StringVar(&cmdFile, "f", "", "image list file")
	flagSet.StringVar(&cmdArch, "a", "amd64,arm64", "architecture list of images, separate with ','")
	flagSet.StringVar(&cmdSource, "s", "", "override the source registry")
	flagSet.StringVar(&cmdDest, "d", "saved-images.tar.gz",
		"Output saved images into destination file " +
		"(can use '-compress' to specify the output file format, default is gzip)")
	flagSet.StringVar(&cmdFailed, "o", "save-failed.txt", "file name of the save failed image list")
	flagSet.StringVar(&cmdCompress, "compress", "gzip",
		"compress format, can be 'gzip', 'zstd' or 'dir'")
	flagSet.BoolVar(&cmdPart, "part", false, "enable segment compress")
	flagSet.StringVar(&cmdPartSize, "part-size", "2G",
		"segment part size (number, or a string with 'K','M','G' suffix)")
	flagSet.BoolVar(&cmdDebug, "debug", false, "enable the debug output")
	flagSet.IntVar(&cmdJobs, "j", 1, "job number, async mode if larger than 1, maximum is 20")
	flagSet.Parse(args)
}

func SaveImages() {
	if cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	var selfCheckFailed = false
	if err := registry.SelfCheckSkopeo(); err != nil {
		logrus.Error("self check skopeo failed.")
		logrus.Error(err)
		selfCheckFailed = true
	}
	if err := registry.SelfCheckDocker(); err != nil {
		logrus.Error("self check docker failed.")
		logrus.Fatal(err)
		selfCheckFailed = true
	}
	if selfCheckFailed {
		os.Exit(1)
	}

	if cmdSource == "" && u.EnvSourceRegistry != "" {
		cmdSource = u.EnvSourceRegistry
	}
	if cmdSource != "" {
		logrus.Infof("Set source registry to [%s]", cmdSource)
	} else {
		logrus.Infof("Set source registry to [%s]", u.DockerHubRegistry)
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
		compressFormat = archive.CompressFormatGzip
	}

	if err := command.ProcessDockerLoginEnv(); err != nil {
		logrus.Error(err)
	}

	if err := u.CheckCacheDirEmpty(); err != nil {
		logrus.Fatal(err)
	}
	// Check cache image directory
	var compressPartSize int = 0
	if compressFormat != archive.CompressFormatDirectory {
		if cmdPart {
			// segment compress enabled
			var err error
			switch {
			case strings.HasSuffix(cmdPartSize, "G"):
				compressPartSize, err = strconv.Atoi(
					strings.TrimSuffix(cmdPartSize, "G"))
				compressPartSize *= archive.GB
			case strings.HasSuffix(cmdPartSize, "g"):
				compressPartSize, err = strconv.Atoi(
					strings.TrimSuffix(cmdPartSize, "g"))
				compressPartSize *= archive.GB
			case strings.HasSuffix(cmdPartSize, "M"):
				compressPartSize, err = strconv.Atoi(
					strings.TrimSuffix(cmdPartSize, "M"))
				compressPartSize *= archive.MB
			case strings.HasSuffix(cmdPartSize, "m"):
				compressPartSize, err = strconv.Atoi(
					strings.TrimSuffix(cmdPartSize, "m"))
				compressPartSize *= archive.MB
			case strings.HasSuffix(cmdPartSize, "K"):
				compressPartSize, err = strconv.Atoi(
					strings.TrimSuffix(cmdPartSize, "K"))
				compressPartSize *= archive.KB
			case strings.HasSuffix(cmdPartSize, "k"):
				compressPartSize, err = strconv.Atoi(
					strings.TrimSuffix(cmdPartSize, "k"))
				compressPartSize *= archive.KB
			default:
				compressPartSize, err = strconv.Atoi(cmdPartSize)
			}
			if err != nil {
				logrus.Fatalf("Failed to get segment part size: %v", err)
			}
			logrus.Infof("Set compress segment part to %q", cmdPartSize)
		}
	}

	var scanner *bufio.Scanner
	var usingStdin bool
	if cmdFile == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE>:<TAG>' from stdin")
		logrus.Info("Use 'Ctrl+D' to exit.")
	} else {
		readFile, err := os.Open(cmdFile)
		if err != nil {
			logrus.Fatal(err)
		}
		defer readFile.Close()
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
	savedTemplate := mirror.NewSavedListTemplate()
	var appendListMutex sync.Mutex
	ch, wg := command.Worker(cmdFailed, func(m *mirror.Mirror) {
		// if image saved successfully
		appendListMutex.Lock()
		savedTemplate.Append(m.GetSavedImageTemplate())
		if usingStdin {
			// Write saved image json
			f := filepath.Join(u.CacheImageDirectory, u.SavedImageListFile)
			u.SaveJson(savedTemplate, f)
			fmt.Printf(">>> ")
		}
		appendListMutex.Unlock()
	})

	var num int = 0
	for scanner.Scan() {
		l := scanner.Text()
		v := processImageListLine(l)
		if len(v) != 2 {
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		num++
		m := mirror.NewMirror(&mirror.MirrorOptions{
			Source:      u.ConstructRegistry(v[0], cmdSource),
			Destination: u.ConstructRegistry(v[0], cmdSource),
			Tag:         v[1],
			Directory:   u.CacheImageDirectory,
			ArchList:    strings.Split(cmdArch, ","),
			Line:        l,
			Mode:        mirror.MODE_SAVE,
			ID:          num,
		})
		if u.GetProjectName(m.Source) == "" {
			m.Source = u.ReplaceProjectName(m.Source, "library")
			m.Destination = u.ReplaceProjectName(m.Destination, "library")
		}
		ch <- m
	}
	close(ch)
	wg.Wait()

	// Write saved image json
	if len(savedTemplate.List) > 0 {
		f := filepath.Join(u.CacheImageDirectory, u.SavedImageListFile)
		u.SaveJson(savedTemplate, f)
	}

	if len(savedTemplate.List) == 0 {
		logrus.Error("No images saved into local directory, skip.")
		os.Exit(1)
	}

	if compressFormat != archive.CompressFormatDirectory {
		logrus.Infof("Compressing %s...", cmdDest)

		err := archive.Compress(
			u.CacheImageDirectory,
			cmdDest,
			compressFormat,
			compressPartSize,
		)
		if err != nil {
			logrus.Fatal(err)
		}
		if !cmdPart {
			// if part compress not enabled,
			// rename file name without .part extension
			if err := os.Rename(cmdDest+".part0", cmdDest); err != nil {
				logrus.Warn(err)
			}
		}
	} else {
		err := os.Rename(u.CacheImageDirectory, cmdDest)
		if err != nil {
			logrus.Warn(err)
		}
	}
	logrus.Infof("Saved images into %q", cmdDest)

	if usingStdin {
		fmt.Println()
	}
}

func processImageListLine(l string) []string {
	var spec []string = make([]string, 0)
	l = strings.TrimSpace(l)
	// Ignore empty/comment line
	if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
		return spec
	}
	// if image name does not have tag, add 'latest'
	if !strings.Contains(l, ":") {
		l = l + ":latest"
	}

	var v []string = make([]string, 0)
	for _, s := range strings.Split(l, ":") {
		if s != "" {
			v = append(v, s)
		}
	}
	if len(v) != 2 {
		return spec
	}
	return v
}
