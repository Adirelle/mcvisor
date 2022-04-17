package events

import "github.com/apex/log"

type (
	Event interface {
		log.Fielder
	}

	Handler interface {
		EventC() chan<- Event
	}
)
