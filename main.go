package main

import (
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/cnrancher/hangar/commands"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        false,
		TimestampFormat: "15:04:05", // hour, time, sec only
		FieldsOrder:     []string{"M_ID", "IMG_ID"},
	})
}

func main() {
	commands.Execute(os.Args[1:])
}
