package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/minecraft"
	"github.com/go-playground/validator/v10"
)

const (
	ConfigFilename = "mcvisor.json"
)

type (
	Config struct {
		Path      string            `json:"-"`
		Minecraft *minecraft.Config `json:"minecraft" validate:"required"`
		Discord   *discord.Config   `json:"discord" validate:"required"`
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
			path = filepath.Join(path, ConfigFilename)
			_, err = os.Stat(path)
		}
		if err == nil {
			return path
		}
	}
	return paths[0]
}

func LoadConfig(path string) (c *Config, err error) {
	c = &Config{
		Path:      path,
		Minecraft: minecraft.NewConfig(filepath.Dir(path)),
		Discord:   &discord.Config{},
	}

	err = c.Read()
	if os.IsNotExist(err) {
		err = c.Write()
	}
	if err != nil {
		return
	}
	err = validator.New().Struct(c)

	return
}

func (c *Config) Apply() {
	c.Discord.Apply()
}

func (c *Config) Read() error {
	content, err := os.ReadFile(c.Path)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, c)
}

func (c *Config) Write() error {
	content, err := json.Marshal(&c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.Path, content, os.FileMode(0o666))
}
