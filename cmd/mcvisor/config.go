package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Adirelle/mcvisor/pkg/minecraft"
)

const (
	ConfigFilename = "mcvisor.json"
)

type (
	Config struct {
		Minecraft *minecraft.Config `json:"minecraft"`
	}
)

func NewConfig() *Config {
	return &Config{Minecraft: &minecraft.Config{}}
}

func (c *Config) Load() error {
	path, err := getConfPath()
	if err != nil {
		return fmt.Errorf("could not get configuration path: %w", err)
	}

	err = c.ReadFrom(path)
	if err != nil {
		return fmt.Errorf("could not read configuration from %s: %w", path, err)
	}

	c.ConfigureDefaults()

	err = c.WriteTo(path)
	if err != nil {
		log.Printf("could not write configuration to %s: %s", path, err)
	}

	c.SetBaseDir(filepath.Dir(path))

	return nil
}

func (c *Config) ConfigureDefaults() {
	c.Minecraft.ConfigureDefaults()
}

func (c *Config) SetBaseDir(baseDir string) {
	c.Minecraft.SetBaseDir(baseDir)
}

func (c *Config) ReadFrom(path string) error {
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
	if err := decoder.Decode(c); err != nil {
		return err
	}
	return nil
}

func (c Config) WriteTo(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&c)
}

func getConfPath() (string, error) {
	var paths []string
	if len(os.Args) >= 2 {
		paths = append(paths, os.Args[1])
	}
	workDir, err := os.Getwd()
	if err == nil {
		paths = append(paths, filepath.Join(workDir, ConfigFilename))
	}
	paths = append(paths, filepath.Join(filepath.Dir(os.Args[0]), ConfigFilename))
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return paths[0], nil
}
