package commands

import (
	"github.com/spf13/cobra/doc"
)

func Doc(dir string) error {
	hangarCmd := newHangarCmd()
	hangarCmd.addCommands()

	header := &doc.GenManHeader{
		Title:   "MINE",
		Section: "1",
	}
	return doc.GenManTree(hangarCmd.cmd, header, dir)
}
