package main

import (
	"os"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/thejerf/suture/v4"
)

type (
	RootSupervisor struct {
		*suture.Supervisor
		Dispatcher *events.AsyncDispatcher
	}
)

func MakeRootSupervisor() RootSupervisor {
	supervisor := suture.NewSimple(os.Args[0])
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
