package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	log.SetLevel(log.DebugLevel)
}

func main() {
	conf, err := LoadConfig(FindConfigFile(ConfigSearchPath()))
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

	server := minecraft.NewServer(*conf.Minecraft, rootSupervisor.Dispatcher)

	pinger := minecraft.NewPinger(*conf.Minecraft, rootSupervisor.Dispatcher)
	rootSupervisor.Add(pinger)

	supervisorCtx, stopSupervisor := context.WithCancel(context.Background())
	control := &serverControl{supervisor: rootSupervisor.Supervisor, server: server, stop: stopSupervisor}
	controller := minecraft.NewController(control, rootSupervisor.Dispatcher)
	rootSupervisor.Add(controller)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-signals
		signal.Stop(signals)
		log.WithField("signal", sig).Info("shutting down on signal")
		controller.SetTarget(minecraft.ShutdownTarget)
		<-time.After(10 * time.Second)
		log.Warn("forcefully shutdown")
		stopSupervisor()
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
