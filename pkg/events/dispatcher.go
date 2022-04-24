package events

import (
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

		mu       *sync.RWMutex
		handlers map[reflect.Type][]reflect.Value
	}

	Subscription struct {
		dispatcher *Dispatcher
		handler    reflect.Value
		eventType  reflect.Type
		cancel     func()
	}
)

var (
	DispatcherCapacity     = 20
	DefaultDispatchTimeout = 10 * time.Second
	HandlerCapacity        = 10

	selectorCasePool = &sync.Pool{
		New: func() any {
			return make([]reflect.SelectCase, 2)
		},
	}
	defaultSelectCase = reflect.SelectCase{Dir: reflect.SelectDefault}
)

func MakeHandler[E any]() chan E {
	return make(chan E, HandlerCapacity)
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		DispatchTimeout: DefaultDispatchTimeout,
		mu:              &sync.RWMutex{},
		handlers:        make(map[reflect.Type][]reflect.Value),
	}
}

func (d *Dispatcher) Dispatch(event any) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	eventValue := reflect.ValueOf(event)
	eventType := eventValue.Type()

	logger := log.WithField("type", eventType.String())
	if fielder, isFielder := event.(log.Fielder); isFielder {
		logger = logger.WithFields(fielder)
	} else {
		logger = logger.WithField("event", fmt.Sprintf("%#v", event))
	}
	logger.Debug("event.dispatch")

	for handlerType, handlers := range d.handlers {
		if !eventType.AssignableTo(handlerType) {
			continue
		}
		for _, handler := range handlers {
			if !dispatchTo(eventValue, handler) {
				logger.WithField("handler", handler.Interface()).Warn("event.dispatch.dropped")
			}
		}
	}
}

func dispatchTo(event reflect.Value, handler reflect.Value) bool {
	cases := selectorCasePool.Get().([]reflect.SelectCase)
	defer selectorCasePool.Put(cases)

	cases[0] = reflect.SelectCase{Dir: reflect.SelectSend, Chan: handler, Send: event}
	cases[1] = defaultSelectCase
	chosen, _, _ := reflect.Select(cases)
	return chosen == 0
}

func (d *Dispatcher) Subscribe(handler interface{}) *Subscription {
	sub := newSubscription(d, handler)
	sub.Apply()
	return sub
}

func newSubscription(dispatcher *Dispatcher, handler interface{}) *Subscription {
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()
	if handlerType.Kind() != reflect.Chan {
		panic("handler must be a channel")
	}

	return &Subscription{
		dispatcher: dispatcher,
		handler:    handlerValue,
		eventType:  handlerType.Elem(),
	}
}

func (s *Subscription) Apply() {
	s.dispatcher.mu.Lock()
	defer s.dispatcher.mu.Unlock()
	s.dispatcher.handlers[s.eventType] = append(s.dispatcher.handlers[s.eventType], s.handler)
}

func (s *Subscription) Cancel() {
	s.dispatcher.mu.Lock()
	defer s.dispatcher.mu.Unlock()
	handlers, found := s.dispatcher.handlers[s.eventType]
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
	s.dispatcher.handlers[s.eventType] = handlers[:j]
}
