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

	dispatcher := event.NewDispatcher()
	rootSupervisor.Add(dispatcher)

	dispatcher.Add(event.HandlerFunc(func(ev event.Event) {
		log.Printf("Event: %s", ev)
	}))

	pinger := minecraft.MakePinger(*conf.Minecraft, dispatcher)
	rootSupervisor.Add(pinger)
	dispatcher.Add(pinger)

	status := minecraft.NewStatusService(dispatcher)
	rootSupervisor.Add(status)
	dispatcher.Add(status)

	bot := discord.NewBot(*conf.Discord, dispatcher)
	rootSupervisor.Add(bot)
	dispatcher.Add(bot)

	server := minecraft.MakeServer(*conf.Minecraft, dispatcher)
	rootSupervisor.Add(server)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	err = rootSupervisor.Serve(ctx)

	if err != nil && err != context.Canceled {
		log.Fatalf("exit reason: %s", err)
	}
}
