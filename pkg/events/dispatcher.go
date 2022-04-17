package events

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/apex/log"
)

type (
	Dispatcher interface {
		Dispatch(Event)
		Add(Handler)
	}

	AsyncDispatcher struct {
		ctl      chan command
		handlers []Handler
	}

	command interface {
		apply(context.Context, *AsyncDispatcher)
	}

	addCommand      struct{ Handler }
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
			cmd.apply(ctx, d)
		case <-ctx.Done():
			return nil
		}
	}
}

func (d *AsyncDispatcher) GoString() string {
	return fmt.Sprintf("Dispatcher(%d, %d/%d)", len(d.handlers), len(d.ctl), cap(d.ctl))
}

func (d AsyncDispatcher) Add(handler Handler) {
	d.ctl <- &addCommand{handler}
}

func (c *addCommand) apply(_ context.Context, d *AsyncDispatcher) {
	d.handlers = append(d.handlers, c.Handler)
}

func (d AsyncDispatcher) Dispatch(events Event) {
	done := make(chan struct{})
	d.ctl <- &dispatchCommand{events, done}
	<-done
}

func (c *dispatchCommand) apply(ctx context.Context, d *AsyncDispatcher) {
	defer close(c.done)

	dispatchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	logger := log.WithFields(c.Event).WithField("type", reflect.TypeOf(c.Event).String())
	logger.Debug("event.dispatch")

	all := &sync.WaitGroup{}
	for _, handler := range d.handlers {
		all.Add(1)

		go func(handler Handler) {
			defer all.Done()
			select {
			case handler.EventC() <- c.Event:
			case <-dispatchCtx.Done():
				logger.WithField("handler", handler).Error("event.dispatch.dropped")
			}
		}(handler)

	}
	all.Wait()
}
