package main

import (
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/logging"
	"github.com/Adirelle/mcvisor/pkg/minecraft"
	"github.com/apex/log"
	"github.com/thejerf/suture/v4"
)

type (
	serverControl struct {
		supervisor *suture.Supervisor
		server     suture.Service
		token      *suture.ServiceToken
		stop       func()
	}
)

var SutureEventLabels = map[suture.EventType]string{
	suture.EventTypeStopTimeout:      "timeout",
	suture.EventTypeServicePanic:     "panic",
	suture.EventTypeServiceTerminate: "terminate",
	suture.EventTypeBackoff:          "backoff",
	suture.EventTypeResume:           "resume",
}

func main() {
	conf, err := LoadConfig(FindConfigFile(ConfigSearchPath()))
	if err != nil {
		stdlog.Fatalf("could not load configuration: %s", err)
	}

	mainSupervisor, dispatcher := NewMainSupervisor(conf)
	mainC := mainSupervisor.ServeBackground(nil)

	minecraftSupervisor := NewMinecraftSupervisor(conf, dispatcher)
	mainSupervisor.Add(minecraftSupervisor)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Kill, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-signals:
			log.WithField("signal", sig).Warn("signal.received")
			dispatcher.Dispatch(minecraft.SystemShutdown)
		case err := <-mainC:
			if err != nil && err != suture.ErrTerminateSupervisorTree {
				stdlog.Fatalf("error: %s", err)
			}
			os.Exit(0)
		}
	}
}

func NewMainSupervisor(conf *Config) (*suture.Supervisor, *events.Dispatcher) {
	supervisor := suture.New("main", suture.Spec{
		EventHook: suture.EventHook(func(event suture.Event) {
			log.
				WithField("message", event.String()).
				WithFields(log.Fields(event.Map())).
				Warnf("suture.%s", SutureEventLabels[event.Type()])
		}),
	})

	SetUpLogging(conf.Logging, supervisor)

	dispatcher := events.NewDispatcher()

	return supervisor, dispatcher
}

func SetUpLogging(conf *logging.Config, supervisor *suture.Supervisor) {
	var service suture.Service
	def := log.Log.(*log.Logger)
	def.Handler, def.Level, service = conf.CreateLogging()
	if service != nil {
		supervisor.Add(service)
	}
}

func NewMinecraftSupervisor(conf *Config, dispatcher *events.Dispatcher) *suture.Supervisor {
	supervisor := suture.NewSimple("minecraft")

	status := minecraft.NewStatusMonitor(dispatcher)
	supervisor.Add(status)

	bot := discord.NewBot(*conf.Discord, dispatcher)
	supervisor.Add(bot)

	server := minecraft.NewServer(conf.Minecraft, dispatcher)

	control := &serverControl{supervisor: supervisor, server: server}
	controller := minecraft.NewController(control, dispatcher)
	supervisor.Add(controller)

	pinger := minecraft.NewPinger(conf.Minecraft.Server, dispatcher)
	supervisor.Add(pinger)

	return supervisor
}

func (c *serverControl) Start() {
	if c.token != nil {
		return
	}
	log.Info("server.enable")
	token := c.supervisor.Add(c.server)
	c.token = &token
}

func (c *serverControl) Stop() {
	if c.token == nil {
		return
	}
	log.Info("server.disable")
	err := c.supervisor.RemoveAndWait(*c.token, 0)
	if err != nil {
		log.WithError(err).Error("server.disable")
	}
	c.token = nil
}

func (c *serverControl) Terminate() {
	log.Info("server.shutdown")
	c.stop()
}
