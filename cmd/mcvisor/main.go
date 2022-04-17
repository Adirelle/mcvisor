package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/minecraft"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
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

var cliLogHandler = cli.New(os.Stderr)

func init() {
	log.SetHandler(cliLogHandler)
	log.SetLevel(log.InfoLevel)
}

func main() {
	conf := NewConfig()
	err := conf.Load()
	if err != nil {
		log.WithError(err).Fatal("could not load configuration")
	}
	conf.Apply()

	rootSupervisor := MakeRootSupervisor()
	rootSupervisor.Add(commands.EventHandler)

	status := minecraft.NewStatusMonitor(rootSupervisor.Dispatcher)
	rootSupervisor.Add(status)

	bot := discord.NewBot(*conf.Discord, rootSupervisor.Dispatcher)
	rootSupervisor.Add(bot)

	serverServices := suture.NewSimple("Server services")

	server := minecraft.NewServer(*conf.Minecraft, rootSupervisor.Dispatcher)
	serverServices.Add(server)

	pinger := minecraft.NewPinger(*conf.Minecraft, rootSupervisor.Dispatcher)
	serverServices.Add(pinger)
	rootSupervisor.Dispatcher.AddHandler(pinger)

	supervisorCtx, stopSupervisor := context.WithCancel(context.Background())
	control := &serverControl{supervisor: rootSupervisor.Supervisor, server: serverServices, stop: stopSupervisor}
	controller := minecraft.NewController(control, rootSupervisor.Dispatcher)
	rootSupervisor.Add(controller)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for signal := range signals {
			log.WithField("signal", signal).Info("signal.received")
			controller.SetTarget(minecraft.ShutdownTarget)
		}
	}()

	err = rootSupervisor.Serve(supervisorCtx)
	if err != nil && err != context.Canceled {
		log.WithError(err).Fatal("exit")
	}
}

func (c *serverControl) Start() {
	if c.token != nil {
		return
	}
	token := c.supervisor.Add(c.server)
	c.token = &token
}

func (c *serverControl) Stop() {
	if c.token == nil {
		return
	}
	err := c.supervisor.RemoveAndWait(*c.token, 0)
	if err != nil {
		log.WithError(err).Error("stopping server")
	}
	c.token = nil
}

func (c *serverControl) Terminate() {
	c.stop()
}
