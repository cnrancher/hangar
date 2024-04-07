package main

import (
	"io"
	"os"
	"runtime"
	"syscall"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/cnrancher/hangar/pkg/commands"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/moby/term"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
)

func main() {
	setup()
	if err := commands.Execute(os.Args[1:]); err != nil {
		cleanup()
		logrus.Fatal(err)
	}
	cleanup()
}

func cleanup() {
	if err := os.RemoveAll(utils.HangarCacheDir()); err != nil {
		logrus.Warnf("failed to delete %q: %v", utils.HangarCacheDir(), err)
	}
	logrus.Debugf("cleanup %q", utils.HangarCacheDir())
}

func setup() {
	formatter := &nested.Formatter{
		HideKeys:        false,
		TimestampFormat: "[15:04:05]", // hour, time, sec only
		FieldsOrder:     []string{"IMG"},
	}
	if !term.IsTerminal(uintptr(syscall.Stdout)) || !term.IsTerminal(uintptr(syscall.Stderr)) {
		// Disable if the output is not terminal.
		formatter.NoColors = true
	}
	logrus.SetFormatter(formatter)
	logrus.SetOutput(io.Discard)
	logrus.AddHook(&writer.Hook{
		// Send logs with level higher than warning to stderr.
		Writer: os.Stderr,
		LogLevels: []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
		},
	})
	logrus.AddHook(&writer.Hook{
		// Send info, debug and trace logs to stdout.
		Writer: os.Stdout,
		LogLevels: []logrus.Level{
			logrus.TraceLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
		},
	})

	if runtime.GOOS == "windows" {
		logrus.Panicf("unsupported OS: %v", runtime.GOOS)
	}
}
