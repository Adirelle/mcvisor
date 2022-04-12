package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/Adirelle/mcvisor/pkg/minecraft"
	"github.com/thejerf/suture/v4"
)

func main() {

	conf := minecraft.Config{}
	conf.ResolvePath("C:\\Users\\gperr\\AppData\\Roaming\\ATLauncher\\instances\\Minecraft1181withFabric")

	svc := minecraft.Service{Config: conf}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	sup := suture.NewSimple("mcvisor")
	sup.Add(svc)

	err := sup.Serve(ctx)
	if (err != nil && err != context.Canceled) {
		log.Fatalf("exit reason: %s", err)
	}
}
