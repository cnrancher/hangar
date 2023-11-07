package manifest

import (
	"context"
	"fmt"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

func PushManifest(
	destName, uname, passwd string, dt []byte,
) error {
	dst := fmt.Sprintf("docker://%s", destName)
	ref, err := alltransports.ParseImageName(dst)
	if err != nil {
		return fmt.Errorf("PushManifest: %w", err)
	}

	skipTls := !cmdconfig.GetBool("tls-verify")
	sysCtx := &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: uname,
			Password: passwd,
		},
		OCIInsecureSkipTLSVerify:    skipTls,
		DockerInsecureSkipTLSVerify: types.NewOptionalBool(skipTls),
	}

	dest, err := ref.NewImageDestination(
		context.Background(), sysCtx)
	if err != nil {
		return fmt.Errorf("PushManifest: %w", err)
	}
	err = dest.PutManifest(
		context.Background(), dt, nil)
	if err != nil {
		return fmt.Errorf("PushManifest: %w", err)
	}
	return nil
}
