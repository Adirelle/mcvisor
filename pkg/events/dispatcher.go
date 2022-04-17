package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apex/log"
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

	dispatchCommand struct {
		Event
		done chan struct{}
	}
)

var DispatchChanCapacity = 20

func NewAsyncDispatcher() *AsyncDispatcher {
	return &AsyncDispatcher{ctl: make(chan command, DispatchChanCapacity)}
}

func (d *AsyncDispatcher) Serve(ctx context.Context) error {
	for {
		select {
		case cmd := <-d.ctl:
			d.handleCommand(cmd, ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (d *AsyncDispatcher) GoString() string {
	return fmt.Sprintf("Dispatcher(%d, %d/%d)", len(d.handlers), len(d.ctl), cap(d.ctl))
}

func (d *AsyncDispatcher) handleCommand(cmd command, ctx context.Context) {
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
		dispatchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		defer close(c.done)
		log.WithFields(c.Event).Debugf("event.dispatch.%s", c.Event.Type())
		all := &sync.WaitGroup{}
		for _, handler := range d.handlers {
			all.Add(1)
			go func(handler Handler) {
				defer all.Done()
				select {
				case handler.EventC() <- c.Event:
				case <-dispatchCtx.Done():
					log.WithField("handler", handler).WithField("event", c.Event).Error("event.dispatch.dropped")
				}
			}(handler)
		}
		all.Wait()
	}
}

func (d AsyncDispatcher) DispatchEvent(events Event) {
	done := make(chan struct{})
	d.ctl <- dispatchCommand{events, done}
	<-done
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
