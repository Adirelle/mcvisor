package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/minecraft"
	"github.com/thejerf/suture/v4"
)

type (
	serverControl struct {
		supervisor *suture.Supervisor
		server     suture.Service
		token      *suture.ServiceToken
	}
)

func main() {
	conf := NewConfig()
	err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}
	conf.Apply()

	rootSupervisor := suture.NewSimple("mcvisor")

	dispatcher := events.NewAsyncDispatcher()
	rootSupervisor.Add(dispatcher)

	dispatcher.AddHandler(commands.EventHandler)
	dispatcher.AddHandler(events.HandlerFunc(LogEvent))

	pinger := minecraft.NewPinger(*conf.Minecraft, dispatcher)
	rootSupervisor.Add(pinger)
	dispatcher.AddHandler(pinger)

	status := minecraft.NewStatusMonitor(dispatcher)
	rootSupervisor.Add(status)
	dispatcher.AddHandler(status)

	bot := discord.NewBot(*conf.Discord, dispatcher)
	rootSupervisor.Add(bot)
	dispatcher.AddHandler(bot)

	server := minecraft.NewServer(*conf.Minecraft, dispatcher)
	control := &serverControl{supervisor: rootSupervisor, server: server}
	controller := minecraft.NewController(control)
	rootSupervisor.Add(controller)

	supervisorCtx, stopSupervisor := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signals
		stopSupervisor()
	}()

	err = rootSupervisor.Serve(supervisorCtx)
	if err != nil && err != context.Canceled {
		log.Fatalf("exit reason: %s", err)
	}
}

func LogEvent(ev events.Event) {
	log.Printf("[%s]: %s", ev.Type(), ev)
}

func (c *serverControl) Start() {
	if c.token != nil {
		return
	}
	log.Printf("starting the server service")
	token := c.supervisor.Add(c.server)
	c.token = &token
}

func (c *serverControl) Stop() {
	if c.token == nil {
		return
	}
	log.Printf("stopping the server service")
	err := c.supervisor.RemoveAndWait(*c.token, 0)
	if err != nil {
		log.Printf("error stopping server: %s", err)
	}
	c.token = nil
}
