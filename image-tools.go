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

func init() {
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        false,
		TimestampFormat: "15:04:05", // hour, time, sec only
		FieldsOrder:     []string{"MID", "IID"},
	})
	logrus.SetOutput(os.Stdout)
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(0)
	}

	// mirror subcmd reads file from image-list txt or stdin and mirror image
	// from source repo to the destination repo
	mirrorCmd := flag.NewFlagSet("mirror", flag.ExitOnError)
	mirrorFile := mirrorCmd.String("f", "", "image list file")
	mirrorArch := mirrorCmd.String("a", "amd64,arm64", "architecture list of images, seperate with ','")
	mirrorSourceReg := mirrorCmd.String("s", "", "override the source registry")
	mirrorDestReg := mirrorCmd.String("d", "", "override the destination registry")
	mirrorJobsReg := mirrorCmd.Int("j", 1, "asynchronous mode if larger than 1, maximun is 20")
	mirrorDebug := mirrorCmd.Bool("debug", false, "enable the debug output")

	switch os.Args[1] {
	case "mirror":
		mirrorCmd.Parse(os.Args[2:])
		if *mirrorDebug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		logrus.Debugf("mirrorFile: %s", *mirrorFile)
		logrus.Debugf("mirrorArch: %s", *mirrorArch)
		logrus.Debugf("sourceReg: %s", *mirrorSourceReg)
		logrus.Debugf("destReg: %s", *mirrorDestReg)
		logrus.Debugf("mirrorJobsReg: %v", *mirrorJobsReg)
		MirrorImages(*mirrorFile, *mirrorArch, *mirrorSourceReg,
			*mirrorDestReg, *mirrorJobsReg)
	case "load": // TODO: load image from tar.gz tarball
	case "save": // TODO: save image to tar.gz tarball with image manifest
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

func MirrorImages(file, arches, srcRegOverride, dstRegOverride string, jobNum int) {
	if dockerUsername == "" || dockerPassword == "" {
		logrus.Fatal("DOCKER_USERNAME, DOCKER_PASSWORD environment not set")
		// TODO: read username and password from stdin
	}

	if srcRegOverride != "" {
		logrus.Infof("Set source registry to [%s]", srcRegOverride)
	} else {
		logrus.Infof("Set source registry to [%s]", u.DockerHubRegistry)
	}

	// Command line parameter is prior than environment variable
	if dstRegOverride == "" && dockerRegistry != "" {
		dstRegOverride = dockerRegistry
	}

	if dstRegOverride != "" {
		logrus.Infof("Set destination registry to [%s]", dstRegOverride)
	} else {
		logrus.Infof("Set destination registry to [%s]", u.DockerHubRegistry)
	}

	// execute docker login command
	err := registry.DockerLogin(dstRegOverride, dockerUsername, dockerPassword)
	if err != nil {
		logrus.Fatalf("MirrorImages login failed: %v", err.Error())
	}

	var scanner *bufio.Scanner
	var usingStdin bool
	if file == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE> <DESTINATION> <TAG>' from stdin")
		logrus.Info("Use 'Ctrl+C' or 'Ctrl+D' to exit.")
	} else {
		readFile, err := os.Open(file)
		if err != nil {
			fmt.Println(err)
		}
		defer readFile.Close()

		scanner = bufio.NewScanner(readFile)
		scanner.Split(bufio.ScanLines)
	}

	if usingStdin && jobNum != 1 {
		logrus.Warn("async mode not supported in stdin mode")
		logrus.Warn("change worker count back to 1")
		jobNum = 1
	}
	if jobNum > 20 {
		logrus.Warn("worker count should be <= 20")
		logrus.Warn("change worker count to 20")
		jobNum = 20
	}
	if jobNum < 1 {
		logrus.Warn("invalid worker count")
		logrus.Warn("change worker count to 1")
		jobNum = 20
	}
	if !usingStdin {
		logrus.Infof("Creating %d job workers", jobNum)
	} else {
		fmt.Printf(">>> ")
	}
	u.MirrorerJobNum = jobNum

	var wg sync.WaitGroup
	// worker function for goroutine pool
	worker := func(id int, ch chan mirror.Mirrorer) {
		defer wg.Done()
		for mirrorer := range ch {
			mirrorer.SetID(fmt.Sprintf("%02d", id))

			logrus.WithField("MID", mirrorer.ID()).
				Infof("SOURCE: [%v] DEST: [%v] TAG: [%v]",
					mirrorer.Source(), mirrorer.Destination(), mirrorer.Tag())

			err := mirrorer.Mirror()
			if err != nil {
				logrus.Errorf("Failed to copy image [%s]", mirrorer.Source())
				logrus.Error(err.Error())
			}
			if usingStdin {
				fmt.Printf(">>> ")
			}
			if mirrorer.Failed() != 0 {
				// TODO: there is some images copy failed in this mirrorer
			}
		}
	}
	mirrorChan := make(chan mirror.Mirrorer)
	for i := 0; i < jobNum; i++ {
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
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		var mirrorer mirror.Mirrorer = mirror.NewMirror(&mirror.MirrorOptions{
			Source:      mirror.ConstructRegistry(v[0], srcRegOverride),
			Destination: mirror.ConstructRegistry(v[1], dstRegOverride),
			Tag:         v[2],
			ArchList:    strings.Split(arches, ","),
		})

		mirrorChan <- mirrorer

	}

	close(mirrorChan)
	wg.Wait()
	if usingStdin {
		fmt.Println()
	}
}
