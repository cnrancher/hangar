package commands

import (
	"fmt"
	"os"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type archiveInitCmd struct {
	*baseCmd
}

func newArchiveInitCmd() *archiveInitCmd {
	cc := &archiveInitCmd{}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "init",
		Short:   "Init an empty Archive file",
		Long:    "Init an empty Archive file for use by sync/archive-store commands.",
		Aliases: []string{"i"},
		Example: `hangar archive init ./ARCHIVE_NAME.zip`,
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cc.run(args); err != nil {
				return err
			}
			return nil
		},
	})

	return cc
}

func (cc *archiveInitCmd) run(args []string) error {
	if len(args) == 0 {
		cc.cmd.Help()
		return fmt.Errorf("archive filename not provided")
	}

	for _, a := range args {
		_, err := os.Stat(a)
		if err == nil {
			return fmt.Errorf("file %q already exists", a)
		}

		aw, err := archive.NewWriter(a)
		if err != nil {
			return fmt.Errorf("failed to create archive writer: %w", err)
		}

		aw.WriteIndex(archive.NewIndex())
		aw.Close()
		logrus.Infof("Create [%v]", a)
	}
	return nil
}
