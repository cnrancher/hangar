package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/image/manifest"
	"github.com/cnrancher/hangar/pkg/utils"
	manifestv5 "github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type mergeManifestCmd struct {
	*baseCmd

	dryRun    bool
	tlsVerify bool
}

func newMergeManifestCmd() *mergeManifestCmd {
	cc := &mergeManifestCmd{}
	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "merge-manifest",
		Short: "Merge multi-arch images manifest index",
		Long:  "",
		Example: `# Merge multi-arch image manifest:
hangar merge-manifest [IMAGE_NAME] [IMAGES]

# Example:
hangar merge-manifest registry.io/library/image:latest \
	registry.io/library/image:amd64 \
	registry.io/library/image:arm64`,
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
	flags.BoolVarP(&cc.dryRun, "dry-run", "", false, "dry run")
	flags.BoolVarP(&cc.tlsVerify, "tls-verify", "", true, "require HTTPS and verify certificates")

	return cc
}

func (cc *mergeManifestCmd) run(args []string) error {
	if len(args) < 3 {
		cc.cmd.Help()
		return fmt.Errorf("merge-manifest command requires at least 3 arguments")
	}

	for i := range args {
		if strings.HasPrefix(args[i], "docker://") {
			continue
		}
		args[i] = fmt.Sprintf("docker://%v", args[i])
	}
	builder, err := manifest.NewBuilder(&manifest.BuilderOpts{
		ReferenceName: args[0],
		SystemContext: &types.SystemContext{
			ArchitectureChoice:          "",
			OSChoice:                    "",
			VariantChoice:               "",
			OCIInsecureSkipTLSVerify:    !cc.tlsVerify,
			DockerInsecureSkipTLSVerify: types.NewOptionalBool(!cc.tlsVerify),
		},
	})
	if err != nil {
		return fmt.Errorf("manifest builder %q: %w", args[0], err)
	}

	ctx := signalContext
	spec := args[1:]
	for _, i := range spec {
		inspector, err := manifest.NewInspector(ctx, &manifest.InspectorOption{
			ReferenceName: i,
			SystemContext: &types.SystemContext{
				ArchitectureChoice:          "",
				OSChoice:                    "",
				VariantChoice:               "",
				OCIInsecureSkipTLSVerify:    !cc.tlsVerify,
				DockerInsecureSkipTLSVerify: types.NewOptionalBool(!cc.tlsVerify),
			},
		})
		if err != nil {
			return fmt.Errorf("inspect image %q: %w", i, err)
		}
		b, mime, err := inspector.Raw(ctx)
		if err != nil {
			return fmt.Errorf("inspect image %q: %w", i, err)
		}
		logrus.Infof("Image %q is %q", i, mime)
		switch mime {
		case manifestv5.DockerV2ListMediaType:
			s2list, err := manifestv5.Schema2ListFromManifest(b)
			if err != nil {
				return fmt.Errorf("failed to load %q manifest: %w", i, err)
			}
			for _, m := range s2list.Manifests {
				mi := manifest.NewImage(m.Digest, m.MediaType, m.Size, nil)
				mi.UpdatePlatform(
					m.Platform.Architecture,
					m.Platform.Variant,
					m.Platform.OS,
					m.Platform.OSVersion,
					m.Platform.OSFeatures,
				)
				builder.Add(mi)
			}
		case manifestv5.DockerV2Schema2MediaType:
			sha256Sum := utils.Sha256Sum(string(b))
			cb, err := inspector.Config(ctx)
			if err != nil {
				return fmt.Errorf("failed to load %q config: %w", i, err)
			}
			m, err := manifestv5.Schema2FromManifest(b)
			if err != nil {
				return fmt.Errorf("failed to load %q manifest: %w", i, err)
			}
			ociConfig := &imgspecv1.Image{}
			err = json.Unmarshal(cb, ociConfig)
			if err != nil {
				return fmt.Errorf("failed to load %q config: %w", i, err)
			}
			mi := manifest.NewImage(
				digest.NewDigestFromEncoded(digest.SHA256, sha256Sum),
				m.MediaType,
				int64(len(b)),
				nil,
			)
			mi.UpdatePlatform(
				ociConfig.Architecture,
				ociConfig.Variant,
				ociConfig.OS,
				ociConfig.OSVersion,
				ociConfig.OSFeatures,
			)
			builder.Add(mi)
		case manifestv5.DockerV2Schema1MediaType,
			manifestv5.DockerV2Schema1SignedMediaType:
			// Schema1 is not supported
			return fmt.Errorf("image %q mime type %q is deprecated and not supported", i, mime)
		case imgspecv1.MediaTypeImageIndex:
			ociIndex := &imgspecv1.Index{}
			err = json.Unmarshal(b, ociIndex)
			if err != nil {
				return fmt.Errorf("failed to load %q manifest: %w", i, err)
			}
			for _, m := range ociIndex.Manifests {
				mi := manifest.NewImage(m.Digest, m.MediaType, m.Size, m.Annotations)
				mi.UpdatePlatform(
					m.Platform.Architecture,
					m.Platform.Variant,
					m.Platform.OS,
					m.Platform.OSVersion,
					m.Platform.OSFeatures,
				)
				builder.Add(mi)
			}
		case imgspecv1.MediaTypeImageManifest:
			m := &imgspecv1.Manifest{}
			err = json.Unmarshal(b, m)
			if err != nil {
				return fmt.Errorf("failed to load %q manifest: %w", i, err)
			}
			sha256Sum := utils.Sha256Sum(string(b))
			cb, err := inspector.Config(ctx)
			if err != nil {
				return fmt.Errorf("failed to load %q config: %w", i, err)
			}
			ociConfig := &imgspecv1.Image{}
			err = json.Unmarshal(cb, ociConfig)
			if err != nil {
				return fmt.Errorf("failed to load %q config: %w", i, err)
			}
			mi := manifest.NewImage(
				digest.NewDigestFromEncoded(digest.SHA256, sha256Sum),
				m.MediaType,
				int64(len(b)),
				m.Annotations,
			)
			mi.UpdatePlatform(
				ociConfig.Architecture,
				ociConfig.Variant,
				ociConfig.OS,
				ociConfig.OSVersion,
				ociConfig.OSFeatures,
			)
			builder.Add(mi)
		}
		inspector.Close()
	}

	if cc.dryRun {
		s, err := builder.String()
		if err != nil {
			return err
		}
		fmt.Printf("%v\n", s)
		return nil
	}
	logrus.Infof("Pushing image manifest index %v with %v contents", args[0], builder.Images())
	return builder.Push(ctx)
}
