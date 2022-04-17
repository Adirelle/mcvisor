package main

import (
	"os"
	"path/filepath"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/apex/log"
	"github.com/thejerf/suture/v4"
)

type (
	RootSupervisor struct {
		*suture.Supervisor
		Dispatcher *events.AsyncDispatcher
	}
)

var SutureEventLabels = map[suture.EventType]string{
	suture.EventTypeStopTimeout:      "timeout",
	suture.EventTypeServicePanic:     "panic",
	suture.EventTypeServiceTerminate: "terminate",
	suture.EventTypeBackoff:          "backoff",
	suture.EventTypeResume:           "resume",
}

func MakeRootSupervisor() RootSupervisor {
	specs := suture.Spec{EventHook: EventHook}
	supervisor := suture.New(filepath.Base(os.Args[0]), specs)
	dispatcher := events.NewAsyncDispatcher()
	supervisor.Add(dispatcher)
	return RootSupervisor{supervisor, dispatcher}
}

func (s RootSupervisor) Add(svc suture.Service) suture.ServiceToken {
	if handler, isHandler := svc.(events.Handler); isHandler {
		s.Dispatcher.AddHandler(handler)
	}
	return s.Supervisor.Add(svc)
}

func EventHook(event suture.Event) {
	log.
		WithField("message", event.String()).
		WithFields(log.Fields(event.Map())).
		Warnf("suture.%s", SutureEventLabels[event.Type()])
}
