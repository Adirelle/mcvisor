package event

import (
	"fmt"
	"time"
)

type (
	Event interface {
		fmt.Stringer
		Type() Type
		When() time.Time
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
