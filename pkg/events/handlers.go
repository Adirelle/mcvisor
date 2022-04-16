package events

type (
	HandlerFunc func(Event)

	HandlerMap map[Type]Handler
)

func (f HandlerFunc) HandleEvent(ev Event) {
	f(ev)
}

func (m HandlerMap) HandleEvent(ev Event) {
	if h, found := m[ev.Type()]; found {
		h.HandleEvent(ev)
	}
}

func MakeOneEventHandler(t Type, h HandlerFunc) HandlerFunc {
	return func(e Event) {
		if e.Type() == t {
			h(e)
		}
	}
}
