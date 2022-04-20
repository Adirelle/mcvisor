package logging

import (
	"github.com/apex/log"
	"github.com/apex/log/handlers/multi"
	"github.com/thejerf/suture/v4"
)

type (
	Config struct {
		Console ConsoleConfig `json:"console"`
		File    *FileConfig   `json:"file,omitempty"`
	}

	factory interface {
		CreateLogging() (log.Handler, log.Level, suture.Service)
	}
)

var _ factory = (*Config)(nil)

func NewConfig() *Config {
	return &Config{
		Console: ConsoleConfig(log.WarnLevel),
		File: &FileConfig{
			Disabled: false,
			Path:     DefaultFilename,
			Level:    log.InfoLevel,
			entries:  make(chan *log.Entry, 100),
		},
	}
}

func (c *Config) CreateLogging() (log.Handler, log.Level, suture.Service) {
	handler, minLevel, svc := c.Console.CreateLogging()

	if c.File != nil && !c.File.Disabled {
		fileHandler, fileLevel, fileSvc := c.File.CreateLogging()
		handler = multi.New(handler, fileHandler)
		if fileLevel < minLevel {
			minLevel = fileLevel
		}
		if svc != nil && fileSvc != nil {
			spv := suture.NewSimple("logger")
			spv.Add(svc)
			spv.Add(fileSvc)
			svc = spv
		}
	}

	return handler, minLevel, svc
}
