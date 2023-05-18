package manifest

import (
	"context"
	"fmt"

	"github.com/cnrancher/hangar/pkg/config"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
)

type BuildManifestListParam struct {
	Digest   string                    `json:"digest"`
	Platform BuildManifestListPlatform `json:"platform"`
}

type BuildManifestListPlatform struct {
	Architecture string `json:"architecture,omitempty"`
	OS           string `json:"os,omitempty"`
	Variant      string `json:"variant,omitempty"`
	OsVersion    string `json:"os.version,omitempty"`
}

func CompareBuildManifests(src, dst []BuildManifestListParam) bool {
	if src == nil || dst == nil {
		return false
	}
	if len(src) != len(dst) {
		return false
	}
	for i := range src {
		if !compareBuildManifest(&src[i], &dst[i]) {
			return false
		}
	}
	return true
}

func compareBuildManifest(src, dst *BuildManifestListParam) bool {
	if src == nil || dst == nil {
		return false
	}
	if src.Digest != dst.Digest {
		return false
	}
	if src.Platform.Architecture != dst.Platform.Architecture {
		return false
	}
	if src.Platform.OS != dst.Platform.OS {
		return false
	}
	if src.Platform.Variant != dst.Platform.Variant {
		return false
	}
	if src.Platform.OsVersion != dst.Platform.OsVersion {
		return false
	}
	return true
}

func BuildManifestExists(
	src []BuildManifestListParam, e BuildManifestListParam) bool {
	if len(src) == 0 {
		return false
	}
	for _, p := range src {
		if compareBuildManifest(&p, &e) {
			return true
		}
	}
	return false
}

func BuildManifestList(
	destImage, uname, passwd string,
	params []BuildManifestListParam,
) (*manifest.Schema2List, error) {
	skipTls := !config.GetBool("tls-verify")
	sysCtx := &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: uname,
			Password: passwd,
		},
		DockerInsecureSkipTLSVerify: types.NewOptionalBool(skipTls),
		OCIInsecureSkipTLSVerify:    skipTls,
	}

	list := manifest.Schema2List{
		SchemaVersion: 2,
		MediaType:     manifest.DockerV2ListMediaType,
		Manifests:     []manifest.Schema2ManifestDescriptor{},
	}

	for _, p := range params {
		dst := fmt.Sprintf("docker://%s@%s", destImage, p.Digest)
		ref, err := alltransports.ParseImageName(dst)
		if err != nil {
			return nil, fmt.Errorf("BuildManifestList: %w", err)
		}
		source, err := ref.NewImageSource(context.Background(), sysCtx)
		if err != nil {
			return nil, fmt.Errorf("BuildManifestList: %w", err)
		}
		dt, mime, err := source.GetManifest(context.Background(), nil)
		if err != nil {
			return nil, fmt.Errorf("BuildManifestList: %w", err)
		}

		m := manifest.Schema2ManifestDescriptor{
			Schema2Descriptor: manifest.Schema2Descriptor{
				MediaType: mime,
				Size:      int64(len(dt)),
				Digest:    digest.Digest(p.Digest),
			},
			Platform: manifest.Schema2PlatformSpec{
				Architecture: p.Platform.Architecture,
				OS:           p.Platform.OS,
				OSVersion:    p.Platform.OsVersion,
				Variant:      p.Platform.Variant,
			},
		}
		list.Manifests = append(list.Manifests, m)
	}
	list.ToSchema2List()
	return &list, nil
}
