package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/event"
	"github.com/Adirelle/mcvisor/pkg/minecraft"
	"github.com/thejerf/suture/v4"
)

func main() {
	conf := NewConfig()
	err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	rootSupervisor := suture.NewSimple("mcvisor")

	dispatcher := event.NewAsyncDispatcher()
	rootSupervisor.Add(dispatcher)

	dispatcher.AddHandler(event.HandlerFunc(func(ev event.Event) {
		log.Printf("[%s]: %s", ev.Type(), ev)
	}))

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
	rootSupervisor.Add(server)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	err = rootSupervisor.Serve(ctx)

	if err != nil && err != context.Canceled {
		log.Fatalf("exit reason: %s", err)
	}
}
