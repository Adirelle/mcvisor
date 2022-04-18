package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/minecraft"
	"github.com/apex/log"
	"github.com/go-playground/validator/v10"
)

const (
	DefaultConfigFilename = "mcvisor.json"
)

type (
	Config struct {
		Path      string            `json:"-"`
		Minecraft *minecraft.Config `json:"minecraft" validate:"required"`
		Discord   *discord.Config   `json:"discord" validate:"required"`
		Logging   *Logging          `json:"logging"`
	}
)

func ConfigSearchPath() []string {
	paths := os.Args[1:]
	workDir, err := os.Getwd()
	if err == nil {
		paths = append(paths, workDir)
	}
	return append(paths, filepath.Dir(os.Args[0]))
}

func FindConfigFile(paths []string) string {
	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			path = filepath.Join(path, DefaultConfigFilename)
			_, err = os.Stat(path)
		}
		if err == nil {
			return path
		}
	}
	return paths[0]
}

func NewConfig(path string) (c *Config) {
	return &Config{
		Path:      path,
		Minecraft: minecraft.NewConfig(filepath.Dir(path)),
		Discord:   discord.NewConfig(),
		Logging:   NewLogging(),
	}
}

func LoadConfig(path string) (c *Config, err error) {
	c = NewConfig(path)

	err = c.Read()
	if err != nil && !os.IsNotExist(err) {
		return
	}

	err = validator.New().Struct(c)
	if err != nil {
		return
	}

	if writeError := c.Write(); writeError != nil {
		log.WithField("path", path).WithError(writeError).Error("log.file.write")
	}

	return
}

func (c *Config) Apply() {
	c.Discord.Apply()
}

func (c *Config) Read() error {
	content, err := os.ReadFile(c.Path)
	if err != nil {
		return fmt.Errorf("could not read configuration: %w", err)
	}
	err = json.Unmarshal(content, c)
	if err != nil {
		return fmt.Errorf("invalid configuration file: %w", err)
	}
	return err
}

func (c *Config) Write() error {
	file, err := os.Create(c.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}
