package saver

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

	// Check cache image directory
	if compressFormat != archive.CompressFormatDirectory {
		if err := u.CheckCacheDirEmpty(); err != nil {
			logrus.Fatal(err)
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
	var writeFileMutex sync.Mutex
	var appendListMutex sync.Mutex
	var wg sync.WaitGroup
	// worker function for goroutine pool
	worker := func(id int, ch chan *mirror.Mirror) {
		defer wg.Done()
		for m := range ch {
			if err := m.StartSave(); err != nil {
				logrus.WithField("M_ID", m.MID).
					Errorf("Failed to save image [%s]", m.Source)
				logrus.WithField("M_ID", m.MID).
					Error(err.Error())
				writeFileMutex.Lock()
				u.AppendFileLine(*cmdFailed,
					fmt.Sprintf("%s:%s", m.Source, m.Tag))
				writeFileMutex.Unlock()
			} else if m.ImageNum()-m.Saved() != 0 {
				// if there are some images save failed in this mirrorer
				logrus.WithField("M_ID", m.MID).
					Errorf("Some images failed to save: %s", m.Source)
				writeFileMutex.Lock()
				u.AppendFileLine(*cmdFailed,
					fmt.Sprintf("%s:%s", m.Source, m.Tag))
				writeFileMutex.Unlock()
			} else {
				// if image saved successfully
				appendListMutex.Lock()
				savedTemplate.Append(m.GetSavedImageTemplate())
				if usingStdin {
					dir := filepath.Join(
						u.CacheImageDirectory, u.SavedImageListFile)
					u.SaveJson(savedTemplate, dir)
				}
				appendListMutex.Unlock()
			}

			if usingStdin {
				fmt.Printf(">>> ")
			}
		}
	}
	mirrorChan := make(chan *mirror.Mirror)
	for i := 0; i < *cmdJobs; i++ {
		wg.Add(1)
		go worker(i+1, mirrorChan)
	}

	var num int = 0
	for scanner.Scan() {
		v := processImageListLine(scanner.Text())
		if len(v) != 2 {
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		num++
		sourceReg := u.GetRegistryName(u.ConstructRegistry(v[0], *cmdSourceReg))
		registryMap := make(map[string]bool)
		if usingStdin && !registryMap[sourceReg] {
			user, passwd, err := registry.GetDockerPassword(sourceReg)
			if err != nil {
				user, passwd, _ = u.ReadUsernamePasswd()
			}
			if err = registry.DockerLogin(sourceReg, user, passwd); err != nil {
				logrus.Warn(err)
			}
			registryMap[sourceReg] = true
		}
		mirrorChan <- mirror.NewMirror(&mirror.MirrorOptions{
			Source:    u.ConstructRegistry(v[0], *cmdSourceReg),
			Tag:       v[1],
			Directory: u.CacheImageDirectory,
			ArchList:  strings.Split(*cmdArch, ","),
			Mode:      mirror.MODE_SAVE,
			ID:        num,
		})
	}
	close(mirrorChan)
	wg.Wait()

	// Write saved image json
	if len(savedTemplate.List) > 0 {
		dir := filepath.Join(u.CacheImageDirectory, u.SavedImageListFile)
		u.SaveJson(savedTemplate, dir)
	}

	if compressFormat != archive.CompressFormatDirectory {
		logrus.Infof("Compressing %s...", *cmdDest)
		err := archive.Compress(u.CacheImageDirectory, *cmdDest, compressFormat)
		if err != nil {
			logrus.Fatal(err)
		}
	} else {
		err := os.Rename(u.CacheImageDirectory, *cmdDest)
		if err != nil {
			logrus.Warn(err)
		}
	}
	logrus.Infof("Successfully saved images into %q", *cmdDest)

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
