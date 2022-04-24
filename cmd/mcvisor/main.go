package main

import (
	"context"
	stdlog "log"
	"os"
	"os/signal"

	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
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

	supervisor := suture.New("main", suture.Spec{
		EventHook: suture.EventHook(func(event suture.Event) {
			log.
				WithField("message", event.String()).
				WithFields(log.Fields(event.Map())).
				Warnf("suture.%s", SutureEventLabels[event.Type()])
		}),
	})
	spvDone := supervisor.ServeBackground(context.Background())

	handler, level, service := conf.Logging.CreateLogging()
	log.SetHandler(handler)
	log.SetLevel(level)
	if service != nil {
		supervisor.Add(service)
	}

	dispatcher := events.NewDispatcher()

	bot := discord.NewBot(*conf.Discord, dispatcher)
	if bot.IsEnabled() {
		supervisor.Add(bot)
		<-bot.Ready()
	}

	server := minecraft.NewServer(conf.Minecraft, dispatcher)
	supervisor.Add(server)

	pinger := minecraft.NewPinger(conf.Minecraft.Server, dispatcher)
	supervisor.Add(pinger)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Kill, os.Interrupt)

	go func() {
		if sig, open := <-signals; open {
			log.WithField("signal", sig).Warn("signal.received")
			server.Shutdown()
		}
	}()

	server.Start()

	err = <-spvDone
	close(signals)
	if err != nil && err != suture.ErrTerminateSupervisorTree {
		stdlog.Fatalf("error: %s", err)
	}
	os.Exit(0)
}
