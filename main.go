package main

import (
	"os"
	"runtime"
	"syscall"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/cnrancher/hangar/commands"
	"github.com/moby/term"
	"github.com/sirupsen/logrus"
)

func init() {
	formatter := &nested.Formatter{
		HideKeys:        false,
		TimestampFormat: "[15:04:05]", // hour, time, sec only
		FieldsOrder:     []string{"IMG"},
	}
	if !term.IsTerminal(uintptr(syscall.Stdout)) {
		formatter.NoColors = true
	}
	logrus.SetFormatter(formatter)

	if runtime.GOOS == "windows" {
		logrus.Panicf("unsupported OS: %v", runtime.GOOS)
	}
}

func main() {
	if err := commands.Execute(os.Args[1:]); err != nil {
		logrus.Error(err)
	}
}
