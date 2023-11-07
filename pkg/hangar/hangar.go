package hangar

import "context"

type Hangar interface {
	Run(ctx context.Context) error
}
