package event

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
		HandleEvent(Event)
	}

	HandlerFunc func(Event)
)

func (f HandlerFunc) HandleEvent(ev Event) {
	f(ev)
}
