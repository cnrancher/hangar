package saver

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"cnrancher.io/image-tools/archive"
	command "cnrancher.io/image-tools/cmd"
	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

var (
	cmd          = flag.NewFlagSet("save", flag.ExitOnError)
	cmdFile      = cmd.String("f", "", "image list file")
	cmdArch      = cmd.String("a", "amd64,arm64", "architecture list of images, separate with ','")
	cmdSourceReg = cmd.String("s", "", "override the source registry")
	cmdDest      = cmd.String("d", "saved-images.tar.gz", "Output saved images into destination file (directory or tar tarball)")
	cmdFailed    = cmd.String("o", "save-failed.txt", "file name of the save failed image list")
	cmdCompress  = cmd.String("compress", "gzip", "compress format, can be 'gzip', 'zstd' or 'dir'")
	cmdPart      = cmd.Bool("part", false, "enable segment compress")
	cmdPartSize  = cmd.String("part-size", "2G", "segment part size (a number, or a string ended with 'K','M' or 'G')")
	cmdDebug     = cmd.Bool("debug", false, "enable the debug output")
	cmdJobs      = cmd.Int("j", 1, "job number, async mode if larger than 1, maximum is 20")
)

func Parse(args []string) {
	cmd.Parse(args)
}

func SaveImages() {
	if *cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if err := registry.SelfCheckSkopeo(); err != nil {
		logrus.Error("registry self check skopeo failed.")
		logrus.Fatal(err)
	} else if err = registry.SelfCheckDocker(); err != nil {
		logrus.Error("registry self check docker failed.")
		logrus.Fatal(err)
	}

	if *cmdSourceReg == "" && u.EnvSourceRegistry != "" {
		*cmdSourceReg = u.EnvSourceRegistry
	}
	if *cmdSourceReg != "" {
		logrus.Infof("Set source registry to [%s]", *cmdSourceReg)
	} else {
		logrus.Infof("Set source registry to [%s]", u.DockerHubRegistry)
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

	if err := command.ProcessDockerLoginEnv(); err != nil {
		logrus.Error(err)
	}

	if err := u.CheckCacheDirEmpty(); err != nil {
		logrus.Fatal(err)
	}
	// Check cache image directory
	var compressPartSize int = 0
	if compressFormat != archive.CompressFormatDirectory {
		if *cmdPart {
			// segment compress enabled
			var err error
			switch {
			case strings.HasSuffix(*cmdPartSize, "G"):
				compressPartSize, err = strconv.Atoi(
					strings.TrimSuffix(*cmdPartSize, "G"))
				compressPartSize *= archive.GB
			case strings.HasSuffix(*cmdPartSize, "M"):
				compressPartSize, err = strconv.Atoi(
					strings.TrimSuffix(*cmdPartSize, "M"))
				compressPartSize *= archive.MB
			case strings.HasSuffix(*cmdPartSize, "K"):
				compressPartSize, err = strconv.Atoi(
					strings.TrimSuffix(*cmdPartSize, "K"))
				compressPartSize *= archive.KB
			default:
				compressPartSize, err = strconv.Atoi(*cmdPartSize)
			}
			if err != nil {
				logrus.Fatalf("Failed to get segment part size: %v", err)
			}
			logrus.Infof("Set compress segment part to %q", *cmdPartSize)
		}
	}

	var scanner *bufio.Scanner
	var usingStdin bool
	if *cmdFile == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE>:<TAG>' from stdin")
		logrus.Info("Use 'Ctrl+D' to exit.")
	} else {
		readFile, err := os.Open(*cmdFile)
		if err != nil {
			logrus.Fatal(err)
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
	u.WorkerNum = *cmdJobs

	u.DeleteIfExist(*cmdFailed)
	savedTemplate := mirror.NewSavedListTemplate()
	var appendListMutex sync.Mutex
	ch, wg := command.Worker(*cmdJobs, *cmdFailed, func(m *mirror.Mirror) {
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
			Source:      u.ConstructRegistry(v[0], *cmdSourceReg),
			Destination: u.ConstructRegistry(v[0], *cmdSourceReg),
			Tag:         v[1],
			Directory:   u.CacheImageDirectory,
			ArchList:    strings.Split(*cmdArch, ","),
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
		logrus.Infof("Compressing %s...", *cmdDest)

		err := archive.Compress(
			u.CacheImageDirectory,
			*cmdDest,
			compressFormat,
			compressPartSize,
		)
		if err != nil {
			logrus.Fatal(err)
		}
		if !*cmdPart {
			// if part compress not enabled,
			// rename file name without .part extension
			if err := os.Rename(*cmdDest+".part0", *cmdDest); err != nil {
				logrus.Warn(err)
			}
		}
	} else {
		err := os.Rename(u.CacheImageDirectory, *cmdDest)
		if err != nil {
			logrus.Warn(err)
		}
	}
	logrus.Infof("Saved images into %q", *cmdDest)

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
