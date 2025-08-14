package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/cnrancher/hangar/pkg/image/manifest"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/types"
	digest "github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	manifestv5 "github.com/containers/image/v5/manifest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type viewOpts struct {
	format string
}

type viewCmd struct {
	*baseCmd
}

func newViewCmd() *viewCmd {
	cc := &viewCmd{}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "view",
		Short: "View SLSA provenance or SBOM data of image",
		Example: `hangar view sbom <IMAGE>
hangar view provenance <IMAGE>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc.cmd.Help()
			return nil
		},
	})

	addCommands(
		cc.cmd,
		newViewProvenanceCmd(),
		newViewSBOMCmd(),
	)

	return cc
}

func getImageAttestationReference(
	ctx context.Context,
	image string,
	osInfo string,
	archInfo string,
	tlsVerify bool,
) (string, error) {
	if osInfo == "" {
		osInfo = runtime.GOOS
	}
	if archInfo == "" {
		archInfo = runtime.GOARCH
	}
	inspector, err := manifest.NewInspector(&manifest.InspectorOption{
		ReferenceName: fmt.Sprintf("docker://%v", image),
		SystemContext: &types.SystemContext{
			ArchitectureChoice:          archInfo,
			OSChoice:                    osInfo,
			VariantChoice:               "",
			OCIInsecureSkipTLSVerify:    !tlsVerify,
			DockerInsecureSkipTLSVerify: types.NewOptionalBool(!tlsVerify),
		},
	})
	if err != nil {
		return "", err
	}
	defer inspector.Close()
	b, mime, err := inspector.Raw(ctx)
	if err != nil {
		return "", err
	}
	manifestSha256sum := utils.Sha256Sum(string(b))
	var imageDigest digest.Digest
	logrus.Debugf("Image %q MediaType is %q", image, mime)
	switch mime {
	case manifestv5.DockerV2ListMediaType:
		list := manifestv5.Schema2List{}
		err = json.Unmarshal(b, &list)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal manifest list: %w", err)
		}
		for _, m := range list.Manifests {
			if m.Platform.Architecture == archInfo && m.Platform.OS == osInfo {
				imageDigest = m.Digest
				logrus.Debugf("image [%v] arch[%v] os[%v] digest[%v]",
					image, archInfo, osInfo, imageDigest)
				return fmt.Sprintf("docker://%v/%v/%v:%v-%v.att",
					utils.GetRegistryName(image),
					utils.GetProjectName(image),
					utils.GetImageName(image),
					imageDigest.Algorithm(),
					imageDigest.Encoded()), nil
			}
		}
		return "", fmt.Errorf("no image found in manifest list of arch %v, OS %v", archInfo, osInfo)
	case imgspecv1.MediaTypeImageManifest,
		manifestv5.DockerV2Schema1MediaType,
		manifestv5.DockerV2Schema1SignedMediaType,
		manifestv5.DockerV2Schema2MediaType:
		return fmt.Sprintf("docker://%v/%v/%v:sha256-%v.att",
			utils.GetRegistryName(image),
			utils.GetProjectName(image),
			utils.GetImageName(image),
			manifestSha256sum), nil
	case imgspecv1.MediaTypeImageIndex:
	default:
		logrus.Errorf("unsupported manifest mediaType: %v", mime)
		return "", fmt.Errorf("unsupported mediaType: %v", mime)
	}

	index := &imgspecv1.Index{}
	err = json.Unmarshal(b, index)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal manifest index: %w", err)
	}

	for _, m := range index.Manifests {
		if m.Platform.Architecture == archInfo && m.Platform.OS == osInfo {
			imageDigest = m.Digest
			logrus.Debugf("image [%v] arch[%v] os[%v] digest[%v]",
				image, archInfo, osInfo, imageDigest)
			break
		}
	}
	if imageDigest == "" {
		return "", fmt.Errorf("no available image for platform '%v/%v'", osInfo, archInfo)
	}

	var attDigest digest.Digest
	for _, m := range index.Manifests {
		if m.Platform.Architecture == "unknown" && m.Platform.OS == "unknown" {
			if len(m.Annotations) == 0 {
				continue
			}
			if m.Annotations["vnd.docker.reference.type"] != "attestation-manifest" {
				continue
			}
			d := m.Annotations["vnd.docker.reference.digest"]
			if d != imageDigest.String() {
				continue
			}
			attDigest = m.Digest
			logrus.Debugf("image [%v] provenance[%v]",
				image, attDigest)
			return fmt.Sprintf("docker://%v/%v/%v@%v",
				utils.GetRegistryName(image),
				utils.GetProjectName(image),
				utils.GetImageName(image),
				attDigest.String()), nil
		}
	}
	return "", fmt.Errorf("no image found in manifest list of arch %v, OS %v", archInfo, osInfo)
}
