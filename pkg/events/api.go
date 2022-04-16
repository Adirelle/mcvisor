package events

import (
	"fmt"
)

type (
	Event interface {
		fmt.Stringer
		Type() Type
		When() Time
	}

	Type string

	Handler interface {
		EventC() chan<- Event
	}
)
