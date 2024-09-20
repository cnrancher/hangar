package private

import (
	"strings"
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
		IsErrorRetryable: func(err error) bool {
			if !retry.IsErrorRetryable(err) {
				return false
			}
			// https://github.com/cnrancher/hangar/issues/44
			// Harbor response non-standard error code, need to detect the
			// error content again to avoid the retry warning message.
			s := err.Error()
			switch {
			case strings.Contains(s, "not found") ||
				strings.Contains(s, "manifest unknow"):
				return false
			case strings.Contains(s, "500 Internal Server Error"):
				return true
			}
			return true
		},
	}
}
