package events

import (
	"fmt"

	"github.com/apex/log"
)

type (
	Event interface {
		fmt.Stringer
		Type() Type
		When() Time
		log.Fielder
	}

	Type string

	Handler interface {
		EventC() chan<- Event
	}
)
