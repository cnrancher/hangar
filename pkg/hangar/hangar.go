package hangar

import (
	"context"
	"errors"
)

var (
	ErrValidateFailed = errors.New("some images failed to validate")
	ErrCopyFailed     = errors.New("some images failed to copy")
	ErrSignFailed     = errors.New("some images failed to sign")
	ErrScanFailed     = errors.New("some images failed to scan")
	ErrInspectFailed  = errors.New("some images failed to inspect")
)

type Hangar interface {
	Run(ctx context.Context) error
	Validate(ctx context.Context) error
	FailedImages() error
}
