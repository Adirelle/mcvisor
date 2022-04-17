package events

type (
	Event interface{}

	Handler interface {
		EventC() chan<- Event
	}
)
