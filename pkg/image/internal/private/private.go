package private

import (
	"time"

	"github.com/containers/common/pkg/retry"
)

const (
	retryMaxTimes = 3
	retryDelay    = time.Millisecond * 100
)

func RetryOptions() *retry.Options {
	return &retry.Options{
		MaxRetry: retryMaxTimes,
		Delay:    retryDelay,
	}
}
