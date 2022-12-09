package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

var (
	saveCmd       = flag.NewFlagSet("save", flag.ExitOnError)
	saveFile      = saveCmd.String("f", "", "image list file")
	saveArch      = saveCmd.String("a", "amd64,arm64", "architecture list of images, seperate with ','")
	saveSourceReg = saveCmd.String("s", "", "override the source registry")
	saveDestDir   = saveCmd.String("d", u.CacheImageDirectory, "specify the output directory")
	saveFailed    = saveCmd.String("o", "save-failed.txt", "file name of the save failed image list")
	saveDebug     = saveCmd.Bool("debug", false, "enable the debug output")
	saveJobs      = saveCmd.Int("j", 1, "job number, async mode if larger than 1, maximum is 20")
)

func SaveImages() {
	if err := registry.SelfCheckSkopeo(); err != nil {
		logrus.Error("registry self check skopeo failed.")
		logrus.Fatal(err)
	} else if err = registry.SelfCheckDocker(); err != nil {
		logrus.Error("registry self check docker failed.")
		logrus.Fatal(err)
	}

	if *saveSourceReg != "" {
		logrus.Infof("Set source registry to [%s]", *saveSourceReg)
	} else {
		logrus.Infof("Set source registry to [%s]", u.DockerHubRegistry)
	}

	// Command line parameter is prior than environment variable
	if *saveDestDir == "" {
		logrus.Panic("destination dir not specified!")
	}

	// Check cache image directory
	ok, err := u.IsDirEmpty(u.CacheImageDirectory)
	if err != nil {
		logrus.Panic(err)
	}
	if !ok {
		logrus.Warnf("Cache folder: '%s' is not empty!", u.CacheImageDirectory)
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Delete it before start save image? [Yes/No] ")
		for {
			text, _ := reader.ReadString('\n')
			if len(text) == 0 {
				continue
			}
			if text[0] == 'Y' || text[0] == 'y' {
				break
			} else {
				logrus.Fatalf("'%s': %v",
					u.CacheImageDirectory, u.ErrDirNotEmpty)
			}
		}
		if err := u.DeleteIfExist(u.CacheImageDirectory); err != nil {
			logrus.Panic(err)
		}
	}
	if err = u.EnsureDirExists(u.CacheImageDirectory); err != nil {
		logrus.Panic(err)
	}

	var scanner *bufio.Scanner
	var usingStdin bool
	if *saveFile == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE>:<TAG>' from stdin")
		logrus.Info("Use 'Ctrl+D' to exit.")
	} else {
		readFile, err := os.Open(*saveFile)
		if err != nil {
			fmt.Println(err)
		}
		defer readFile.Close()

		scanner = bufio.NewScanner(readFile)
		scanner.Split(bufio.ScanLines)
	}

	u.CheckWorkerNum(usingStdin, saveJobs)
	if !usingStdin {
		logrus.Infof("Creating %d job workers", *saveJobs)
	} else {
		fmt.Printf(">>> ")
	}
	u.MirrorerJobNum = *saveJobs

	u.DeleteIfExist(*saveFailed)
	savedTemplate := mirror.NewSavedListTemplate()
	var writeFileMutex sync.Mutex
	var appendListMutex sync.Mutex
	var wg sync.WaitGroup
	// worker function for goroutine pool
	worker := func(id int, ch chan mirror.Mirrorer) {
		defer wg.Done()
		for m := range ch {
			m.SetID(fmt.Sprintf("%02d", id))

			logrus.WithField("M_ID", m.ID()).
				Infof("SOURCE: [%v] TAG: [%v]", m.Source(), m.Tag())

			err := m.StartSave()
			if err != nil {
				logrus.WithField("M_ID", m.ID()).
					Errorf("Failed to save image [%s]", m.Source())
				logrus.WithField("M_ID", m.ID()).
					Error(err.Error())
				writeFileMutex.Lock()
				u.AppendFileLine(*saveFailed,
					fmt.Sprintf("%s:%s\n", m.Source(), m.Tag()))
				writeFileMutex.Unlock()
			} else if m.ImageNum()-m.Saved() != 0 {
				// if there are some images save failed in this mirrorer
				logrus.WithField("M_ID", m.ID()).
					Errorf("Some images failed to save: %s", m.Source())
				writeFileMutex.Lock()
				u.AppendFileLine(*saveFailed,
					fmt.Sprintf("%s:%s\n", m.Source(), m.Tag()))
				writeFileMutex.Unlock()
			}
			appendListMutex.Lock()
			savedTemplate.Append(m.GetSavedImageTemplate())
			if usingStdin {
				u.SaveJson(savedTemplate,
					filepath.Join(*saveDestDir, u.SavedImageListFile))
			}
			appendListMutex.Unlock()

			if usingStdin {
				fmt.Printf(">>> ")
			}
		}
	}
	mirrorChan := make(chan mirror.Mirrorer)
	for i := 0; i < *saveJobs; i++ {
		wg.Add(1)
		go worker(i+1, mirrorChan)
	}

	for scanner.Scan() {
		l := scanner.Text()
		l = strings.TrimSpace(l)
		// Ignore empty/comment line
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
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
			logrus.Errorf("Invalid line format")
			logrus.Errorf("Should be: '<SOURCE>:<TAG>'")
			logrus.Debugf("Skip line %s", l)
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		var mirrorer mirror.Mirrorer = mirror.NewMirror(&mirror.MirrorOptions{
			Source:    mirror.ConstructRegistry(v[0], *saveSourceReg),
			Tag:       v[1],
			Directory: *saveDestDir,
			ArchList:  strings.Split(*saveArch, ","),
			Mode:      mirror.MODE_SAVE,
		})

		mirrorChan <- mirrorer
	}
	close(mirrorChan)
	wg.Wait()

	// Write saved image json
	u.SaveJson(savedTemplate, filepath.Join(*saveDestDir, u.SavedImageListFile))
	// TODO: create tar.gz tarball

	if usingStdin {
		fmt.Println()
	}
}
