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
	Done <-chan struct{}

	Dispatcher struct {
		DispatchTimeout time.Duration

		ctl      chan command
		handlers map[reflect.Type][]reflect.Value
	}

	Subscription struct {
		handler   reflect.Value
		eventType reflect.Type
		cancel    func()
	}

	command interface {
		Apply(*Dispatcher, context.Context)
	}

	subscribe struct {
		*Subscription
		Done chan struct{}
	}

	unsubscribe struct {
		*Subscription
		Done chan struct{}
	}

	dispatch struct {
		Event interface{}
		Done  chan struct{}
	}
)

var (
	DispatcherCapacity     = 20
	DefaultDispatchTimeout = 10 * time.Second
	HandlerCapacity        = 10
)

func MakeHandler[E any]() chan E {
	return make(chan E, HandlerCapacity)
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		DispatchTimeout: DefaultDispatchTimeout,
		ctl:             make(chan command, DispatcherCapacity),
		handlers:        make(map[reflect.Type][]reflect.Value),
	}
}

func (d *Dispatcher) Serve(ctx context.Context) error {
	for {
		select {
		case cmd := <-d.ctl:
			cmd.Apply(d, ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (d *Dispatcher) Subscribe(handler interface{}) *Subscription {
	sub := newSubscription(handler)
	sub.cancel = func() {
		cmd := &unsubscribe{sub, make(chan struct{})}
		d.ctl <- cmd
		<-cmd.Done
	}

	cmd := &subscribe{sub, make(chan struct{})}
	d.ctl <- cmd
	<-cmd.Done
	return sub
}

func (d *Dispatcher) Dispatch(event interface{}) Done {
	cmd := &dispatch{event, make(chan struct{})}
	d.ctl <- cmd
	return cmd.Done
}

func newSubscription(handler interface{}) *Subscription {
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()
	if handlerType.Kind() != reflect.Chan {
		panic("handler must be a channel")
	}

	return &Subscription{
		handler:   handlerValue,
		eventType: handlerType.Elem(),
	}
}

func (s *Subscription) Cancel() {
	s.cancel()
}

func (s *subscribe) Apply(d *Dispatcher, _ context.Context) {
	defer close(s.Done)
	d.handlers[s.eventType] = append(d.handlers[s.eventType], s.handler)
}

func (s *unsubscribe) Apply(d *Dispatcher, _ context.Context) {
	defer close(s.Done)
	handlers, found := d.handlers[s.eventType]
	if !found {
		return
	}

	j := 0
	for i, l := 0, len(handlers); i < l; i++ {
		if h := handlers[i]; h != s.handler {
			handlers[j] = h
			j++
		}
	}
	d.handlers[s.eventType] = handlers[:j]
}

func (c *dispatch) Apply(d *Dispatcher, ctx context.Context) {
	defer close(c.Done)

	dispatchCtx, cancel := context.WithTimeout(ctx, d.DispatchTimeout)
	defer cancel()
	doneChan := reflect.ValueOf(dispatchCtx.Done())

	eventValue := reflect.ValueOf(c.Event)
	eventType := eventValue.Type()

	logger := log.WithField("type", eventType.String())
	if fielder, isFielder := c.Event.(log.Fielder); isFielder {
		logger = logger.WithFields(fielder)
	} else {
		logger = logger.WithField("event", fmt.Sprintf("%#v", c.Event))
	}
	logger.Debug("event.dispatch")

	sync := &sync.WaitGroup{}
	for handlerType, handlers := range d.handlers {
		if eventType.AssignableTo(handlerType) {
			for _, handler := range handlers {
				sync.Add(1)
				go func(handler reflect.Value) {
					defer sync.Done()
					chosen, _, _ := reflect.Select([]reflect.SelectCase{
						{Dir: reflect.SelectSend, Chan: handler, Send: eventValue},
						{Dir: reflect.SelectRecv, Chan: doneChan},
					})
					if chosen != 0 {
						logger.WithField("handler", handler.Interface()).Warn("event.dispatch.dropped")
					}
				}(handler)
			}
		}
	}
	sync.Wait()
}
