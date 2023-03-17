package commands

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/cnrancher/hangar/pkg/config"
	"github.com/cnrancher/hangar/pkg/mirror"
	"github.com/cnrancher/hangar/pkg/registry"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	checkDocker = 0x01
	checkBuildx = 0x02
	checkSkopeo = 0x04

	checkDockerBuildx = checkDocker | checkBuildx
	checkDockerSkopeo = checkDocker | checkSkopeo

	checkAll = checkDocker | checkBuildx | checkSkopeo
)

type baseCmd struct {
	cmd *cobra.Command

	workerCallback func(m *mirror.Mirror) error
	workerChan     chan *mirror.Mirror
	failedList     []string
	wg             sync.WaitGroup
}

func (cc *baseCmd) getCommand() *cobra.Command {
	return cc.cmd
}

// check docker, docker-buildx, skopeo are installed or not
func (cc *baseCmd) selfCheckDependencies(flag uint8) error {
	ni := make([]string, 0, 3)
	if flag&checkDocker > 0 {
		if err := registry.SelfCheckDocker(); err != nil {
			ni = append(ni, "docker")
		}
	}
	if flag&checkBuildx > 0 {
		if err := registry.SelfCheckBuildX(); err != nil {
			ni = append(ni, "docker-buildx")
		}
	}
	if flag&checkSkopeo > 0 {
		if err := registry.SelfCheckSkopeo(); err != nil {
			ni = append(ni, "skopeo")
		}
	}
	if len(ni) != 0 {
		b := strings.Builder{}
		for i := range ni {
			b.WriteString(ni[i])
			b.WriteString(" ")
		}
		return fmt.Errorf("some dependencies not installed: %q", b.String())
	}
	return nil
}

func (cc *baseCmd) processDockerLogin() error {
	if utils.EnvSourcePassword != "" && utils.EnvSourceUsername != "" {
		logrus.Infof("running docker login to source registry")
		err := registry.DockerLogin(
			utils.EnvSourceRegistry,
			utils.EnvSourceUsername,
			utils.EnvSourcePassword)
		if err != nil {
			return fmt.Errorf("failed to login to %s: %w",
				utils.EnvSourceRegistry, err)
		}
	}

	if utils.EnvDestPassword != "" && utils.EnvDestUsername != "" {
		logrus.Infof("running docker login to destination registry")
		err := registry.DockerLogin(
			utils.EnvDestRegistry,
			utils.EnvDestUsername,
			utils.EnvDestPassword)
		if err != nil {
			return fmt.Errorf("failed to login to %s: %w",
				utils.EnvDestRegistry, err)
		}
	}

	return nil
}

func (cc *baseCmd) prepareImageCacheDirectory() error {
	ok, err := utils.IsDirEmpty(utils.CacheImageDirectory)
	if err != nil {
		logrus.Panic(err)
	}
	if !ok {
		logrus.Warnf("cache folder: '%s' is not empty!",
			utils.CacheImageDirectory)
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("delete it before start save/load image? [y/N] ")
		for {
			text, _ := reader.ReadString('\n')
			if len(text) == 0 {
				continue
			}
			if text[0] == 'Y' || text[0] == 'y' {
				break
			} else {
				return fmt.Errorf("'%s': %w",
					utils.CacheImageDirectory, utils.ErrDirNotEmpty)
			}
		}
		if err := utils.DeleteIfExist(utils.CacheImageDirectory); err != nil {
			return err
		}
	}
	if err = utils.EnsureDirExists(utils.CacheImageDirectory); err != nil {
		return err
	}
	return nil
}

func (cc *baseCmd) runDockerLogin(reg string) error {
	logrus.Infof("logging into %q", reg)
	username, passwd, err := registry.GetDockerPassword(reg)
	if err != nil {
		logrus.Warnf("please input password of registry %q manually", reg)
		username, passwd, _ = utils.ReadUsernamePasswd()
	}
	if err := registry.DockerLogin(reg, username, passwd); err != nil {
		return err
	}
	if err := registry.SkopeoLogin(reg, username, passwd); err != nil {
		return err
	}
	return nil
}

func (cc *baseCmd) prepareWorker() {
	workerNum := config.GetInt("jobs")
	if workerNum > utils.MAX_WORKER_NUM {
		logrus.Warnf("worker count should be <= %v", utils.MAX_WORKER_NUM)
		logrus.Warnf("change worker count to %v", utils.MAX_WORKER_NUM)
		workerNum = utils.MAX_WORKER_NUM
		config.Set("jobs", workerNum)
	} else if workerNum < utils.MIN_WORKER_NUM {
		logrus.Warnf("invalid worker count: %v", workerNum)
		logrus.Warnf("change worker count to %v", utils.MIN_WORKER_NUM)
		workerNum = utils.MIN_WORKER_NUM
		config.Set("jobs", workerNum)
	}

	mu := sync.RWMutex{}
	worker := func() {
		defer cc.wg.Done()
		for m := range cc.workerChan {
			err := m.Start()
			if err != nil {
				logrus.WithField("M_ID", m.MID).
					Errorf("FAILED [%s:%s]", m.Source, m.Tag)
				logrus.WithField("M_ID", m.MID).Error(err)
				mu.Lock()
				cc.failedList = append(cc.failedList, m.Line)
				sort.Strings(cc.failedList)
				mu.Unlock()
			}
			if cc.workerCallback != nil {
				cc.workerCallback(m)
			}
		}
	}
	cc.workerChan = make(chan *mirror.Mirror)
	for i := 0; i < config.GetInt("jobs"); i++ {
		cc.wg.Add(1)
		go worker()
	}
}

func (cc *baseCmd) finish() {
	close(cc.workerChan)
	cc.wg.Wait()

	fName := config.GetString("failed")
	if len(cc.failedList) > 0 {
		utils.SaveSlice(fName, cc.failedList)
	} else {
		utils.DeleteIfExist(fName)
	}
}

type cmder interface {
	getCommand() *cobra.Command
}

func addCommands(root *cobra.Command, commands ...cmder) {
	for _, command := range commands {
		cmd := command.getCommand()
		if cmd == nil {
			continue
		}
		root.AddCommand(cmd)
	}
}
