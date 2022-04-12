package event

import "time"

type (
	HandlerFunc func(Event)
	HandlerStack []Handler
)

func (f HandlerFunc) HandleEvent(ev Event) {
	f(ev)
}

func (f HandlerFunc) And(h Handler) Handler {
	if stack, isStack := h.(HandlerStack); isStack {
		return stack.And(f)
	} else {
		return HandlerStack([]Handler{f, h})
	}
}

func (s HandlerStack) HandleEvent(ev Event) {
	for _, h := range s {
		h.HandleEvent(ev)
	}
}

func (s HandlerStack) And(h Handler) Handler {
	return HandlerStack(append(s, h))
}

func And(hs ...Handler) Handler {
	s := hs[0]
	for _, h := range hs[1:] {
		s = s.And(h)
	}
	return s
}

func FormatTime(when time.Time) string {
	return when.Format("2006-01-02 15:04:05")
}
