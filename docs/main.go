package main

import (
	"os"

	"github.com/cnrancher/hangar/pkg/commands"
	"github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) < 2 {
		logrus.Fatalf("Usage: %v <PATH>", os.Args[0])
	}
	if err := commands.Doc(os.Args[1]); err != nil {
		logrus.Fatal(err)
	}
}
