package private

import (
	"strings"
	"time"

	"github.com/containers/common/pkg/retry"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	retryMaxTimes = 3
	retryDelay    = time.Millisecond * 100
)

func RetryOptions() *retry.Options {
	return &retry.Options{
		MaxRetry: retryMaxTimes,
		Delay:    retryDelay,
		IsErrorRetryable: func(err error) bool {
			s := err.Error()
			switch {
			case strings.Contains(s, "not found") ||
				strings.Contains(s, "manifest unknow") ||
				strings.Contains(s, "no such file"):
				return false
			}

			if retry.IsErrorRetryable(err) {
				return true
			}

			// Workaround to retry for some timeout/server error
			switch {
			case strings.Contains(s, "500 Internal Server Error") ||
				strings.Contains(s, "timeout") ||
				strings.Contains(s, "stopped after 10 redirects") ||
				strings.Contains(s, "reset by peer"):
				return true
			}
			return false
		},
	}
}

func IsAttestations(m *imgspecv1.Descriptor) bool {
	if m == nil {
		return false
	}
	if m.Platform.Architecture != "unknown" {
		return false
	}
	if m.Platform.OS != "unknown" {
		return false
	}
	if len(m.Annotations) == 0 {
		return false
	}
	if m.Annotations["vnd.docker.reference.type"] != "attestation-manifest" {
		return false
	}
	return true
}
