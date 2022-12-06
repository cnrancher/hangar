package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

var (
	dockerUsername = os.Getenv("DOCKER_USERNAME")
	dockerPassword = os.Getenv("DOCKER_PASSWORD")
	dockerRegistry = os.Getenv("DOCKER_REGISTRY")
)

// mirror COMMAND reads file from image-list txt or stdin, then mirror images
// from source repo to the destination repo
var (
	mirrorCmd       = flag.NewFlagSet("mirror", flag.ExitOnError)
	mirrorFile      = mirrorCmd.String("f", "", "image list file")
	mirrorArch      = mirrorCmd.String("a", "amd64,arm64", "architecture list of images, seperate with ','")
	mirrorSourceReg = mirrorCmd.String("s", "", "override the source registry")
	mirrorDestReg   = mirrorCmd.String("d", "", "override the destination registry")
	mirrorFailed    = mirrorCmd.String("o", "mirror-failed.txt", "file name of the mirror failed image list")
	mirrorDebug     = mirrorCmd.Bool("debug", false, "enable the debug output")
	mirrorJobs      = mirrorCmd.Int("j", 1, "job number, async mode if larger than 1, maximun is 20")
)

var (
	saveCmd       = flag.NewFlagSet("save", flag.ExitOnError)
	saveFile      = saveCmd.String("f", "", "image list file")
	saveArch      = saveCmd.String("a", "amd64,arm64", "architecture list of images, seperate with ','")
	saveSourceReg = saveCmd.String("s", "", "override the source registry")
	saveDestDir   = saveCmd.String("d", "./output/", "specify the output directory")
	saveFailed    = saveCmd.String("o", "save-failed.txt", "file name of the save failed image list")
	saveDebug     = saveCmd.Bool("debug", false, "enable the debug output")
	saveJobs      = saveCmd.Int("j", 1, "job number, async mode if larger than 1, maximum is 20")
)

func init() {
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        false,
		TimestampFormat: "15:04:05", // hour, time, sec only
		FieldsOrder:     []string{"M_ID", "S_ID", "IMG_ID"},
	})
	logrus.SetOutput(os.Stdout)
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "mirror":
		mirrorCmd.Parse(os.Args[2:])
		if *mirrorDebug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		logrus.Debugf("saveFile: %s", *saveFile)
		logrus.Debugf("saveArch: %s", *saveArch)
		logrus.Debugf("sourceReg: %s", *mirrorSourceReg)
		logrus.Debugf("destReg: %s", *mirrorDestReg)
		logrus.Debugf("mirrorJobs: %v", *mirrorJobs)
		logrus.Debugf("mirrorFailed: %v", *mirrorFailed)
		MirrorImages()
	case "load": // TODO: load image from tar.gz tarball
	case "save":
		saveCmd.Parse(os.Args[2:])
		if *saveDebug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		logrus.Debugf("mirrorFile: %s", *mirrorFile)
		logrus.Debugf("mirrorArch: %s", *mirrorArch)
		logrus.Debugf("saveSourceReg: %s", *saveSourceReg)
		logrus.Debugf("saveDestDir: %s", *saveDestDir)
		logrus.Debugf("saveFailed: %v", *saveFailed)
		logrus.Debugf("saveJobs: %v", *saveJobs)
		SaveImages()
	default:
		showHelp()
		os.Exit(0)
	}
}

func showHelp() {
	fmt.Printf("Usage:\t%s COMMAND [OPTIONS]\n", os.Args[0])
	fmt.Println()
	fmt.Printf("Run '%s COMMAND --help' for more information on a command.\n", os.Args[0])
	fmt.Println()
	fmt.Printf("Commands: \n")
	fmt.Printf("  mirror \tMirror image from source registry to destination registry.\n")
	fmt.Printf("  load \t\tWIP.\n")
	fmt.Printf("  save \t\tWIP.\n")
}

func MirrorImages() {
	if err := registry.SelfCheck(); err != nil {
		logrus.Error("registry self check failed.")
		logrus.Fatal(err)
	}

	if dockerUsername == "" || dockerPassword == "" {
		logrus.Fatal("DOCKER_USERNAME, DOCKER_PASSWORD environment not set")
		// TODO: read username and password from stdin
	}

	if *mirrorSourceReg != "" {
		logrus.Infof("Set source registry to [%s]", *mirrorSourceReg)
	} else {
		logrus.Infof("Set source registry to [%s]", u.DockerHubRegistry)
	}

	// Command line parameter is prior than environment variable
	if *mirrorDestReg == "" && dockerRegistry != "" {
		*mirrorDestReg = dockerRegistry
	}

	if *mirrorDestReg != "" {
		logrus.Infof("Set destination registry to [%s]", *mirrorDestReg)
	} else {
		logrus.Infof("Set destination registry to [%s]", u.DockerHubRegistry)
	}

	// execute docker login command
	err := registry.DockerLogin(*mirrorDestReg, dockerUsername, dockerPassword)
	if err != nil {
		logrus.Fatalf("MirrorImages login failed: %v", err.Error())
	}

	var scanner *bufio.Scanner
	var usingStdin bool
	if *mirrorFile == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE> <DESTINATION> <TAG>' from stdin")
		logrus.Info("Use 'Ctrl+C' or 'Ctrl+D' to exit.")
	} else {
		readFile, err := os.Open(*mirrorFile)
		if err != nil {
			fmt.Println(err)
		}
		defer readFile.Close()

		scanner = bufio.NewScanner(readFile)
		scanner.Split(bufio.ScanLines)
	}

	// output copy failed image list into failed list txt
	failedImageListFile, err := os.OpenFile(*mirrorFailed,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		logrus.Errorf("Failed to open file: %s", *mirrorFailed)
		logrus.Fatal(err.Error())
	}
	defer failedImageListFile.Close()

	if usingStdin && *mirrorJobs != 1 {
		logrus.Warn("async mode not supported in stdin mode")
		logrus.Warn("change worker count back to 1")
		*mirrorJobs = 1
	}
	if *mirrorJobs > 20 {
		logrus.Warn("worker count should be <= 20")
		logrus.Warn("change worker count to 20")
		*mirrorJobs = 20
	}
	if *mirrorJobs < 1 {
		logrus.Warn("invalid worker count")
		logrus.Warn("change worker count to 1")
		*mirrorJobs = 20
	}
	if !usingStdin {
		logrus.Infof("Creating %d job workers", *mirrorJobs)
	} else {
		fmt.Printf(">>> ")
	}
	u.MirrorerJobNum = *mirrorJobs

	var writeFileMutex sync.Mutex
	var wg sync.WaitGroup
	// worker function for goroutine pool
	worker := func(id int, ch chan mirror.Mirrorer) {
		defer wg.Done()
		for m := range ch {
			m.SetID(fmt.Sprintf("%02d", id))

			logrus.WithField("M_ID", m.ID()).
				Infof("SOURCE: [%v] DEST: [%v] TAG: [%v]",
					m.Source(), m.Destination(), m.Tag())

			err := m.StartMirror()
			if err != nil {
				logrus.WithField("M_ID", m.ID()).
					Errorf("Failed to copy image [%s]", m.Source())
				logrus.WithField("M_ID", m.ID()).
					Error("Mirror", err.Error())
				writeFileMutex.Lock()
				failedImageListFile.WriteString(
					fmt.Sprintf("%s %s %s\n",
						m.Source(), m.Destination(), m.Tag()))
				failedImageListFile.Sync()
				writeFileMutex.Unlock()
			} else if m.ImageNum()-m.Copied() != 0 {
				// if there are some images copy failed in this mirrorer
				logrus.WithField("M_ID", m.ID()).
					Errorf("Some images failed to mirror: %s", m.Source())
				writeFileMutex.Lock()
				failedImageListFile.WriteString(
					fmt.Sprintf("%s %s %s\n",
						m.Source(), m.Destination(), m.Tag()))
				failedImageListFile.Sync()
				writeFileMutex.Unlock()
				// TODO: sort file
			}
			if usingStdin {
				fmt.Printf(">>> ")
			}
		}
	}
	mirrorChan := make(chan mirror.Mirrorer)
	for i := 0; i < *mirrorJobs; i++ {
		wg.Add(1)
		go worker(i+1, mirrorChan)
	}

	for scanner.Scan() {
		l := scanner.Text()
		// Ignore empty/comment line
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		var v []string = make([]string, 0, 4)
		for _, s := range strings.Split(l, " ") {
			if s != "" {
				v = append(v, s)
			}
		}
		if len(v) != 3 {
			logrus.Errorf("Invalid line format")
			logrus.Errorf("Should be: '<SOURCE> <DESTINATION> <TAG>'")
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		var mirrorer mirror.Mirrorer = mirror.NewMirror(&mirror.MirrorOptions{
			Source:      mirror.ConstructRegistry(v[0], *mirrorSourceReg),
			Destination: mirror.ConstructRegistry(v[1], *mirrorDestReg),
			Tag:         v[2],
			ArchList:    strings.Split(*mirrorArch, ","),
		})

		mirrorChan <- mirrorer

	}

	close(mirrorChan)
	wg.Wait()
	if usingStdin {
		fmt.Println()
	}
}

func SaveImages() {
	if err := registry.SelfCheck(); err != nil {
		logrus.Error("registry self check failed.")
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

	var scanner *bufio.Scanner
	var usingStdin bool
	if *saveFile == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE>:<TAG>' from stdin")
		logrus.Info("Use 'Ctrl+C' or 'Ctrl+D' to exit.")
	} else {
		readFile, err := os.Open(*saveFile)
		if err != nil {
			fmt.Println(err)
		}
		defer readFile.Close()

		scanner = bufio.NewScanner(readFile)
		scanner.Split(bufio.ScanLines)
	}

	// output copy failed image list into failed list txt
	failedImageListFile, err := os.OpenFile(*saveFailed,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		logrus.Errorf("Failed to open file: %s", *saveFailed)
		logrus.Fatal(err.Error())
	}
	defer failedImageListFile.Close()

	if usingStdin && *saveJobs != 1 {
		logrus.Warn("async mode not supported in stdin mode")
		logrus.Warn("change worker count back to 1")
		*saveJobs = 1
	}
	if *saveJobs > 20 {
		logrus.Warn("worker count should be <= 20")
		logrus.Warn("change worker count to 20")
		*saveJobs = 20
	}
	if *saveJobs < 1 {
		logrus.Warn("invalid worker count")
		logrus.Warn("change worker count to 1")
		*saveJobs = 20
	}
	if !usingStdin {
		logrus.Infof("Creating %d job workers", *saveJobs)
	} else {
		fmt.Printf(">>> ")
	}
	u.MirrorerJobNum = *saveJobs

	var writeFileMutex sync.Mutex
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
					Error("Mirror", err.Error())
				writeFileMutex.Lock()
				failedImageListFile.WriteString(
					fmt.Sprintf("%s:%s\n", m.Source(), m.Tag()))
				failedImageListFile.Sync()
				writeFileMutex.Unlock()
			} else if m.ImageNum()-m.Saved() != 0 {
				// if there are some images save failed in this mirrorer
				logrus.WithField("M_ID", m.ID()).
					Errorf("Some images failed to save: %s", m.Source())
				writeFileMutex.Lock()
				failedImageListFile.WriteString(
					fmt.Sprintf("%s:%s\n",
						m.Source(), m.Tag()))
				failedImageListFile.Sync()
				writeFileMutex.Unlock()
				// TODO: sort file
			}
			if usingStdin {
				fmt.Printf(">>> ")
			}
		}
	}
	mirrorChan := make(chan mirror.Mirrorer)
	for i := 0; i < *mirrorJobs; i++ {
		wg.Add(1)
		go worker(i+1, mirrorChan)
	}

	for scanner.Scan() {
		l := scanner.Text()
		// Ignore empty/comment line
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
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
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		var mirrorer mirror.Mirrorer = mirror.NewMirror(&mirror.MirrorOptions{
			Source:    mirror.ConstructRegistry(v[0], *saveSourceReg),
			Tag:       v[1],
			Directory: *saveDestDir,
			ArchList:  strings.Split(*mirrorArch, ","),
		})

		mirrorChan <- mirrorer

	}

	close(mirrorChan)
	wg.Wait()
	if usingStdin {
		fmt.Println()
	}
}
