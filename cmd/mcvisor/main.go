package main

import (
	"context"
	"log"
	"os"
	"os/signal"

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

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	rootSupervisor := suture.NewSimple("mcvisor")

	dispatcher := event.NewDispatcher()
	rootSupervisor.Add(dispatcher)

	dispatcher.Add(event.HandlerFunc(func(ev event.Event) {
		log.Printf("Event: %s", ev)
	}))

	var pingerToken suture.ServiceToken
	pinger := minecraft.MakePinger(*conf.Minecraft, dispatcher)

	dispatcher.Add(event.HandlerFunc(func(ev event.Event) {
		switch ev.(type) {
		case minecraft.ServerStartedEvent:
			pingerToken = rootSupervisor.Add(pinger)
		case minecraft.ServerStoppedEvent:
			rootSupervisor.Remove(pingerToken)
		}
	}))

	server := minecraft.MakeServer(*conf.Minecraft, dispatcher)
	rootSupervisor.Add(server)

	bot := discord.NewBot(*conf.Discord, dispatcher)
	rootSupervisor.Add(bot)
	dispatcher.Add(bot)

	err = rootSupervisor.Serve(ctx)
	if (err != nil && err != context.Canceled) {
		log.Fatalf("exit reason: %s", err)
	}
}
