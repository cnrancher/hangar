package hangar

import (
	"fmt"

	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/source"
)

type Error struct {
	id          int
	e           error
	source      *source.Source
	destination *destination.Destination
}

func NewError(id int, e error, s *source.Source, d *destination.Destination) error {
	return &Error{
		id:          id,
		e:           e,
		source:      s,
		destination: d,
	}
}

func (e *Error) Error() string {
	if e.source == nil {
		return fmt.Sprintf("error occurred on [IMG: %d]: %v",
			e.id, e.e)
	}
	if e.destination == nil {
		return fmt.Sprintf("error occurred on [IMG: %d] [%v]: %v",
			e.id, e.source.ReferenceNameWithoutTransport(), e.e)
	}

	return fmt.Sprintf("error occurred on [IMG: %d] [%v] => [%v]: %v",
		e.id, e.source.ReferenceNameWithoutTransport(),
		e.destination.ReferenceNameWithoutTransport(),
		e.e)
}
