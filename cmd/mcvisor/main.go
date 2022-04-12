package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/Adirelle/mcvisor/pkg/event"
	"github.com/Adirelle/mcvisor/pkg/minecraft"
	"github.com/thejerf/suture/v4"
)

func main() {

	conf, err := loadConfig()
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
	pinger := minecraft.MakePinger(conf, dispatcher)

	dispatcher.Add(event.HandlerFunc(func(ev event.Event) {
		switch ev.(type) {
		case minecraft.ServerStartedEvent:
			pingerToken = rootSupervisor.Add(pinger)
		case minecraft.ServerStoppedEvent:
			rootSupervisor.Remove(pingerToken)
		}
	}))

	server := minecraft.MakeServer(conf, dispatcher)
	rootSupervisor.Add(server)

	err = rootSupervisor.Serve(ctx)
	if (err != nil && err != context.Canceled) {
		log.Fatalf("exit reason: %s", err)
	}
}

func loadConfig() (minecraft.Config, error) {
	conf := minecraft.Config{}
	path, err := getConfPath()
	if err != nil {
		return conf, fmt.Errorf("could not get configuration path: %w", err)
	}

	err = readConfigFrom(path, &conf)
	if err != nil {
		return conf, fmt.Errorf("could not read configuration from %s: %w", path, err)
	}

	conf.ConfigureDefaults()

	err = writeConfigTo(path, conf)
	if err != nil {
		log.Printf("could not write configuration to %s: %s", path, err)
	}

	conf.SetBaseDir(filepath.Dir(path))

	return conf, nil
}

func readConfigFrom(path string, conf *minecraft.Config) error {
	log.Printf("reading configuration from %s", path)
	var file *os.File
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		log.Printf("configuration file %s does not exist", path)
		return nil
	} else if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(conf); err != nil {
		return err
	}
	return nil
}

func writeConfigTo(path string, conf minecraft.Config) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(conf)
}

func getConfPath() (string, error) {
	var paths []string
	if len(os.Args) >= 2 {
		paths = append(paths, os.Args[1])
	}
	workDir, err := os.Getwd()
	if err == nil {
		paths = append(paths, filepath.Join(workDir, "mcvisor.json"))
	}
	paths = append(paths, filepath.Join(filepath.Dir(os.Args[0]), "mcvisor.json"))
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return paths[0], nil
}
