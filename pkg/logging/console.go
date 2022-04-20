package logging

import (
	"encoding/json"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/level"
	"github.com/thejerf/suture/v4"
)

type (
	ConsoleConfig log.Level
)

var (
	_ factory          = (*ConsoleConfig)(nil)
	_ json.Marshaler   = (*ConsoleConfig)(nil)
	_ json.Unmarshaler = (*ConsoleConfig)(nil)
)

func (c ConsoleConfig) CreateLogging() (log.Handler, log.Level, suture.Service) {
	return level.New(cli.Default, log.Level(c)), log.Level(c), nil
}

func (c ConsoleConfig) MarshalJSON() ([]byte, error) {
	return log.Level(c).MarshalJSON()
}

func (c *ConsoleConfig) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*log.Level)(c))
}
