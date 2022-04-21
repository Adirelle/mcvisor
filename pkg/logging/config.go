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

func NewConfig(baseDir string) *Config {
	return &Config{
		Console: ConsoleConfig(log.WarnLevel),
		File:    NewFileConfig(baseDir),
	}
}

func (c *Config) CreateLogging() (handler log.Handler, minLevel log.Level, service suture.Service) {
	var handlers []log.Handler
	var services []suture.Service
	minLevel = log.FatalLevel

	for _, factory := range []factory{c.Console, c.File} {
		handler, level, service := factory.CreateLogging()
		if handler == nil {
			continue
		}
		handlers = append(handlers, handler)
		if level < minLevel {
			minLevel = level
		}
		if service != nil {
			services = append(services, service)
		}
	}

	handler = reduce(handlers, log.Log.(*log.Logger).Handler, combineHandlers)
	service = reduce(services, nil, comineServices)

	return
}

func reduce[T any](values []T, def T, combine func([]T) T) T {
	switch len(values) {
	case 0:
		return def
	case 1:
		return values[0]
	default:
		return combine(values)
	}
}

func combineHandlers(handlers []log.Handler) log.Handler {
	return multi.New(handlers...)
}

func comineServices(services []suture.Service) suture.Service {
	spv := suture.NewSimple("loggers")
	for _, service := range services {
		spv.Add(service)
	}
	return spv
}
