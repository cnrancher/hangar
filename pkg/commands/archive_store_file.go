package commands

import (
	"fmt"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type storeFile struct {
	*baseCmd
}

func newStoreFileCmd() *storeFile {
	cc := &storeFile{}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "file",
		Short:   "store file in Hangar archive file",
		Long:    "",
		Example: ``,
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

	flags := cc.baseCmd.cmd.Flags()
	_ = flags

	return cc
}

func (cc *storeFile) run(args []string) error {
	if len(args) == 0 {
		cc.cmd.Help()
		return fmt.Errorf("file not provided")
	}
	return nil
}
