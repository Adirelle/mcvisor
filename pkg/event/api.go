package event

import "fmt"

type (
	Event     fmt.Stringer

	Handler interface {
		HandleEvent(Event)
	}

	HandlerFunc func(Event)
)

func (f HandlerFunc) HandleEvent(ev Event) {
	f(ev)
}
