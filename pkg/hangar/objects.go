package hangar

import (
	"time"

	hangarcopy "github.com/cnrancher/hangar/pkg/copy"
	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/source"
)

// copyObject is the object for sending to worker pool when copying image
type copyObject struct {
	image       string
	source      *source.Source
	destination *destination.Destination

	timeout time.Duration

	id     int
	copier *hangarcopy.Copier
}
