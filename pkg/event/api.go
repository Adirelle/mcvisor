package event

import "fmt"

type (
	Event     fmt.Stringer

	Handler interface {
		HandleEvent(Event)
		And(Handler) Handler
	}
)
