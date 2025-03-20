package commands

import (
	"fmt"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/hangar/archive/oci"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type storeChartOpts struct {
	file      string
	name      string
	version   string
	tlsVerify bool
}

type storeChartCmd struct {
	*baseCmd
	*storeChartOpts
}

func newStoreChartCmd() *storeChartCmd {
	cc := &storeChartCmd{
		storeChartOpts: new(storeChartOpts),
	}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "chart",
		Short:   "Store OCI format Helm Chart in archive",
		Aliases: []string{"charts", "c"},
		Long:    "",
		Example: `# Add OCI Helm Chart to archive file
hangar archive store chart \
	--file saved_images.zip \
	oci://example.com/chart \
	--name NAME --version VERSION

# Add multiple tgz Helm Charts to archive file
hangar archive store chart \
	--file saved_images.zip \
	./path/to/chart1.tgz \
	./path/to/chart2.tgz

# Add Helm Chart from HTTP Helm Repo
hangar archive store chart \
	--file saved_images.zip \
	https://example.com/chart/repo \
	--name NAME --version VERSION`,
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
	flags.StringVarP(&cc.file, "file", "f", "", "Path to the Hangar archive file (zip)")
	flags.StringVarP(&cc.version, "version", "v", "", "Chart version (optional)")
	flags.StringVarP(&cc.name, "name", "n", "", "Chart name of the helm repository")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "Require HTTPS and verify certificates")

	return cc
}

func (cc *storeChartCmd) run(args []string) error {
	if len(args) == 0 {
		cc.cmd.Help()
		return fmt.Errorf("helm chart not provided")
	}
	if cc.file == "" {
		return fmt.Errorf("archive file not provided")
	}

	policy, err := cc.getPolicy()
	if err != nil {
		return fmt.Errorf("failed to get policy: %w", err)
	}
	au, err := archive.NewUpdater(cc.file)
	if err != nil {
		return err
	}
	defer au.Close()

	for _, a := range args {
		chart := oci.NewChart(&oci.ChartOptions{
			CommonOpts: oci.CommonOpts{
				InsecureSkipVerify: !cc.tlsVerify,
				SystemContext:      cc.baseCmd.newSystemContext(),
				Policy:             policy,
			},
			URL:     a,
			Name:    cc.name,
			Version: cc.version,
		})
		if err := chart.Fetch(signalContext); err != nil {
			return fmt.Errorf("failed to fetch %q, name %q, version %q: %w",
				a, cc.name, cc.version, err)
		}
		if err := chart.WriteArchive(au); err != nil {
			return fmt.Errorf("failed to write chart %q to archive: %w",
				chart.CacheDir(), err)
		}
		logrus.Infof("Add OCI Helm Chart [%v]", chart.Source())
	}
	return nil
}
