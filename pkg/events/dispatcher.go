package events

import (
	"context"
	"fmt"
)

type (
	Dispatcher interface {
		DispatchEvent(Event)
	}

	AsyncDispatcher struct {
		ctl      chan command
		handlers []Handler
	}

	command interface{}

	addCommand    struct{ Handler }
	removeCommand struct{ Handler }

	dispatchCommand struct{ Event }
)

var DispatchChanCapacity = 20

func NewAsyncDispatcher() *AsyncDispatcher {
	return &AsyncDispatcher{ctl: make(chan command, DispatchChanCapacity)}
}

func (d *AsyncDispatcher) Serve(ctx context.Context) error {
	for {
		select {
		case cmd := <-d.ctl:
			d.handleCommand(cmd)
		case <-ctx.Done():
			return nil
		}
	}
}

func (d *AsyncDispatcher) GoString() string {
	return fmt.Sprintf("Dispatcher(%d, %d/%d)", len(d.handlers), len(d.ctl), cap(d.ctl))
}

func (d *AsyncDispatcher) handleCommand(cmd command) {
	switch c := cmd.(type) {
	case addCommand:
		d.handlers = append(d.handlers, c.Handler)
	case removeCommand:
		for i, handler := range d.handlers {
			if handler == c.Handler {
				d.handlers = append(d.handlers[:i], d.handlers[i+1:]...)
				break
			}
		}
	case dispatchCommand:
		for _, handler := range d.handlers {
			handler.EventC() <- c.Event
		}

	}
}

func (d AsyncDispatcher) DispatchEvent(events Event) {
	d.ctl <- dispatchCommand{events}
}

func (d AsyncDispatcher) HandleEvent(events Event) {
	d.DispatchEvent(events)
}

func (d AsyncDispatcher) AddHandler(handler Handler) {
	d.ctl <- addCommand{handler}
}

func (d AsyncDispatcher) RemoveHandler(handler Handler) {
	d.ctl <- removeCommand{handler}
}
