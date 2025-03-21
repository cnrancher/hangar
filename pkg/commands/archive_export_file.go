package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/hangar/archive/oci"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/registry"

	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type exportFileOpts struct {
	file    string // archive file name
	name    string // custom file name
	autoYes bool
}

type exportFileCmd struct {
	*baseCmd
	*exportFileOpts
}

func newExportFileCmd() *exportFileCmd {
	cc := &exportFileCmd{
		exportFileOpts: &exportFileOpts{},
	}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:     "file",
		Short:   "Extract the custom file from Hangar Archive",
		Aliases: []string{"f"},
		Long:    "",
		Example: `# Extract the custom file from archive file
hangar archive export file \
	--name FILENAME \
	--file ARCHIVE.zip`,
		PreRun: func(cmd *cobra.Command, args []string) {
			utils.SetupLogrus(cc.hideLogTime)
			if cc.debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debugf("Debug output enabled")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cc.run(); err != nil {
				return err
			}
			return nil
		},
	})

	flags := cc.baseCmd.cmd.Flags()
	flags.StringVarP(&cc.file, "file", "f", "", "hangar archive file")
	flags.StringVarP(&cc.name, "name", "n", "", "file name to be extracted from archive file")
	flags.BoolVarP(&cc.autoYes, "auto-yes", "y", false, "answer yes automatically (used in shell script)")

	return cc
}

func (cc *exportFileCmd) run() error {
	if cc.file == "" {
		cc.cmd.Help()
		return fmt.Errorf("hangar archive file not provided")
	}
	if cc.name == "" {
		cc.cmd.Help()
		return fmt.Errorf("file name not provided")
	}
	if err := utils.CheckFileExistsPrompt(signalContext, cc.name, cc.autoYes); err != nil {
		return err
	}

	ar, err := archive.NewReader(cc.file)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", cc.file, err)
	}
	defer ar.Close()

	b, err := ar.Index()
	if err != nil {
		return fmt.Errorf("failed to load index from archive: %w", err)
	}
	index := archive.NewIndex()
	err = index.Unmarshal(b)
	if err != nil {
		return fmt.Errorf("failed to decode index: %v", err)
	}
	expectedSource := fmt.Sprintf("%v/%v/%v",
		utils.DockerHubRegistry, oci.DefaultFileProject, cc.name)
	var expectedImage *archive.Image
	for _, image := range index.List {
		if image.Source == expectedSource && image.Tag == utils.DefaultTag &&
			len(image.ArchList) == 0 && len(image.OsList) == 0 {
			expectedImage = image
			break
		}
	}
	if expectedImage == nil {
		return fmt.Errorf("failed to find custom file %q from archive %v: %w",
			cc.name, cc.file, os.ErrNotExist)
	}
	// Validate OCI image to be extracted
	if len(expectedImage.Images) == 0 {
		return fmt.Errorf("no OCI image found")
	}
	spec := expectedImage.Images[0]
	if spec.MediaType != imgspecv1.MediaTypeImageManifest {
		logrus.Warnf("Image %q does is not a custom file OCI image", expectedSource)
		return fmt.Errorf("unexpected image mediaType %q, should be %q",
			spec.MediaType, imgspecv1.MediaTypeImageManifest)
	}
	manifestLayerPath := filepath.Join(archive.SharedBlobDir, "sha256", spec.Digest.Encoded())
	f, err := ar.LoadFile(manifestLayerPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest %q from archive: %w", manifestLayerPath, err)
	}
	defer f.Close()

	if b, err = io.ReadAll(f); err != nil {
		return fmt.Errorf("failed to read manifest %q from archive: %w", manifestLayerPath, err)
	}
	manifest := &imgspecv1.Manifest{}
	if err := json.Unmarshal(b, manifest); err != nil {
		return fmt.Errorf("failed to decode manifest: %w", err)
	}
	if len(manifest.Layers) == 0 {
		return fmt.Errorf("image %q does not have layers", expectedSource)
	}
	layer := manifest.Layers[0]
	if layer.MediaType != oci.FileLayerMediaType {
		logrus.Warnf("Layer %q mediaType is %q",
			layer.Digest, layer.MediaType)
		if layer.MediaType == registry.ChartLayerMediaType || layer.MediaType == registry.ProvLayerMediaType {
			logrus.Warnf("Image %q is a Helm Chart, not a Custom File", expectedSource)
			logrus.Warnf("Use 'helm pull oci://<IMAGE>' command to pull OCI Helm Charts")
		}
		return fmt.Errorf("unexpected image layer mediaType: %q, should be %q",
			layer.MediaType, oci.FileLayerMediaType)
	}
	fileLayerPath := filepath.Join(archive.SharedBlobDir, "sha256", layer.Digest.Encoded())
	layerFile, err := ar.LoadFile(fileLayerPath)
	if err != nil {
		return fmt.Errorf("failed to load layer file %q from archive: %w", fileLayerPath, err)
	}
	defer layerFile.Close()
	file, err := os.Create(cc.name)
	if err != nil {
		return fmt.Errorf("failed to create %q: %w", cc.name, err)
	}
	defer file.Close()

	if _, err = io.Copy(file, layerFile); err != nil {
		return fmt.Errorf("failed to write %q: %w", file.Name(), err)
	}
	logrus.Infof("Load [%v]", file.Name())

	return nil
}
