package validator

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	command "cnrancher.io/image-tools/cmd"
	"cnrancher.io/image-tools/mirror"
	"cnrancher.io/image-tools/registry"
	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

var (
	cmd          = flag.NewFlagSet("mirror-validate", flag.ExitOnError)
	cmdFile      = cmd.String("f", "", "image list file")
	cmdArch      = cmd.String("a", "amd64,arm64", "architecture list of images, separate with ','")
	cmdSourceReg = cmd.String("s", "", "override the source registry")
	cmdDestReg   = cmd.String("d", "", "override the destination registry")
	cmdFailed    = cmd.String("o", "mirror-validate-failed.txt", "file name of the validate failed image list")
	cmdDebug     = cmd.Bool("debug", false, "enable the debug output")
	cmdJobs      = cmd.Int("j", 1, "job number, async mode if larger than 1, maximun is 20")
)

func Parse(args []string) {
	cmd.Parse(args)
}

func ValidateImages() {
	if *cmdDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if err := registry.SelfCheckSkopeo(); err != nil {
		logrus.Error("registry self check skopeo failed.")
		logrus.Fatal(err)
	}
	// Command line parameter is prior than environment variable
	if *cmdDestReg == "" && u.EnvDestRegistry != "" {
		*cmdDestReg = u.EnvDestRegistry
	}
	if *cmdSourceReg == "" && u.EnvSourceRegistry != "" {
		*cmdSourceReg = u.EnvSourceRegistry
	}
	logrus.Debugf("Source registry %q", *cmdSourceReg)
	logrus.Debugf("Dest registry %q", *cmdDestReg)
	if err := command.ProcessDockerLoginEnv(); err != nil {
		logrus.Warn(err)
	}

	var scanner *bufio.Scanner
	var usingStdin bool
	if *cmdFile == "" {
		// read line from stdin
		scanner = bufio.NewScanner(os.Stdin)
		usingStdin = true
		logrus.Info("Reading '<SOURCE> <DESTINATION> <TAG>' from stdin")
		logrus.Info("Use 'Ctrl+C' or 'Ctrl+D' to exit.")
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

	ch, wg := command.Worker(*cmdJobs, *cmdFailed, func(m *mirror.Mirror) {
		if usingStdin {
			fmt.Printf(">>> ")
		}
	})
	var num int = 0
	for scanner.Scan() {
		l := scanner.Text()
		v := processImageListLine(l)
		if len(v) != 3 {
			if usingStdin {
				fmt.Printf(">>> ")
			}
			continue
		}

		num++
		m := mirror.NewMirror(&mirror.MirrorOptions{
			Source:      u.ConstructRegistry(v[0], *cmdSourceReg),
			Destination: u.ConstructRegistry(v[1], *cmdDestReg),
			Tag:         v[2],
			ArchList:    strings.Split(*cmdArch, ","),
			Line:        l,
			Mode:        mirror.MODE_MIRROR_VALIDATE,
			ID:          num,
		})
		ch <- m
	}

	close(ch)
	wg.Wait()
	if usingStdin {
		fmt.Println()
	}
}

func processImageListLine(l string) []string {
	var spec []string = make([]string, 0)
	if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
		return spec
	}
	var v []string = make([]string, 0, 4)
	for _, s := range strings.Split(l, " ") {
		if s != "" {
			v = append(v, s)
		}
	}
	if len(v) != 3 {
		return spec
	}
	return v
}
