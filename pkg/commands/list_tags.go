package commands

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cnrancher/hangar/pkg/image/manifest"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type listTagsCmd struct {
	*baseCmd

	tlsVerify bool
}

func newListTagsCmd() *listTagsCmd {
	cc := &listTagsCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "list-tags IMAGE_NAME",
		Aliases: []string{},
		Short:   "List image tags in the registry server",
		Long:    "",
		Example: `hangar list-tags IMAGE_NAME`,
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
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")

	return cc
}

func (cc *listTagsCmd) run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("image reference not provided")
	}
	if !strings.HasPrefix(args[0], "docker:") {
		args[0] = fmt.Sprintf("docker://%s", args[0])
	}

	ctx := signalContext
	inspector, err := manifest.NewInspector(&manifest.InspectorOption{
		ReferenceName: args[0],
		SystemContext: &types.SystemContext{
			OCIInsecureSkipTLSVerify:    !cc.tlsVerify,
			DockerInsecureSkipTLSVerify: types.NewOptionalBool(!cc.tlsVerify),
		},
	})
	if err != nil {
		return err
	}
	defer inspector.Close()

	tags, err := inspector.Tags(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}
	slices.Sort(tags)
	fmt.Printf("%v\n", utils.ToJSON(tags))
	return nil
}
