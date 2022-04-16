package events

import (
	"context"

	"github.com/thejerf/suture/v4"
)

type (
	HandlerBase chan Event

	HandlerFunc func(Event)

	FuncHandler struct {
		HandlerBase
		handler HandlerFunc
	}
)

var HandlerChanCapacity = 50

func MakeHandlerBase() HandlerBase {
	return HandlerBase(make(chan Event, HandlerChanCapacity))
}

func (b HandlerBase) EventC() chan<- Event {
	return b
}

func MakeHandler(handler HandlerFunc) FuncHandler {
	return FuncHandler{MakeHandlerBase(), handler}
}

func (f FuncHandler) Serve(ctx context.Context) error {
	return Serve(f.HandlerBase, f.handler, ctx)
}

func Serve(events <-chan Event, handler HandlerFunc, ctx context.Context) error {
	for {
		select {
		case event, open := <-events:
			if open {
				handler(event)
			} else {
				return suture.ErrDoNotRestart
			}
		case <-ctx.Done():
			return nil
		}
	}
}
