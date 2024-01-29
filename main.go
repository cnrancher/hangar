package main

import (
	"io"
	"os"
	"runtime"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/cnrancher/hangar/commands"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"golang.org/x/term"
)

func init() {
	formatter := &nested.Formatter{
		HideKeys:        false,
		TimestampFormat: "15:04:05", // hour, time, sec only
		FieldsOrder:     []string{"M_ID", "IMG_ID"},
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
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

func main() {
	commands.Execute(os.Args[1:])
}
