package main

import (
	"os"
	"syscall"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/cnrancher/hangar/commands"
	"github.com/moby/term"
	"github.com/sirupsen/logrus"
)

func init() {
	formatter := &nested.Formatter{
		HideKeys:        false,
		NoFieldsSpace:   true,
		TimestampFormat: "[15:04:05]", // hour, time, sec only
		FieldsOrder:     []string{"IMG"},
	}
	if !term.IsTerminal(uintptr(syscall.Stdout)) {
		formatter.NoColors = true
	}
	logrus.SetFormatter(formatter)
}

func main() {
	commands.Execute(os.Args[1:])
}
