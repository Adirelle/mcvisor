package event

import (
	"context"
	"time"
)

type (
	Dispatcher struct {
		ctl chan command
		handlers []Handler
	}

	command interface {}

	addCommand struct { Handler }
	removeCommand struct { Handler }
	dispatchCommand struct { Event }
)

func NewDispatcher() *Dispatcher {
	return &Dispatcher{ctl: make(chan command, 5)}
}

func (d Dispatcher) Serve(ctx context.Context) error {
	for {
		select {
		case cmd := <- d.ctl:
			d.handleCommand(cmd)
		case <- ctx.Done():
			return nil
		}
	}
}

func (d *Dispatcher) handleCommand(cmd command) {
	switch c := cmd.(type) {
	case addCommand:
		d.handlers = append(d.handlers, c.Handler)
	case removeCommand:
		for i, handler := range d.handlers {
			if handler == c.Handler {
				d.handlers = append(d.handlers[:i], d.handlers[i+1:]...)
				return
			}
		}
	case dispatchCommand:
		for _, handler := range d.handlers {
			handler.HandleEvent(c.Event)
		}
	}
}

func (d Dispatcher) HandleEvent(event Event) {
	d.ctl <- dispatchCommand{event}
}

func (d Dispatcher) Add(handler Handler) {
	d.ctl <- addCommand{handler}
}

func (d Dispatcher) Remove(handler Handler) {
	d.ctl <- removeCommand{handler}
}

func FormatTime(when time.Time) string {
	return when.Format("2006-01-02 15:04:05")
}
