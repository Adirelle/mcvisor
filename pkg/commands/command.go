package commands

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/apex/log"
)

type (
	Name string

	Definition struct {
		Name        Name
		Description string
		permissions.Category
	}

	Command struct {
		*Definition
		events.Time
		permissions.Actor
		Arguments []string
		Reply     *bufio.Writer
	}
)

const CommandType events.Type = "command"

var (
	Prefix rune = '!'

	ErrNoCommandPrefix = errors.New("missing command prefix")
	ErrUnknownCommand  = errors.New("unknown command")

	_ events.Event = Command{}
)

func (n Name) String() string {
	return fmt.Sprintf("%c%s", Prefix, string(n))
}

func (Command) Type() events.Type {
	return CommandType
}

func (c Command) String() string {
	return fmt.Sprintf("%s %v", c.Name, c.Arguments)
}

func (c Command) IsAllowed() bool {
	return c.Permission().Allow(c.Actor)
}

func (c Command) Fields() log.Fields {
	return map[string]interface{}{
		"command": c.Name,
		"args":    c.Arguments,
		"actor":   c.Actor.DescribeActor(),
	}
}

func Parse(line string) (cmd Command, err error) {
	if line[0] != byte(Prefix) {
		err = ErrNoCommandPrefix
		return
	}

	parts := strings.Split(line[1:], " ")
	name := Name(parts[0])
	def, found := commands[name]
	if !found {
		err = ErrUnknownCommand
		return
	}

	cmd.Time = events.Now()
	cmd.Definition = &def
	cmd.Arguments = parts[1:]
	return
}
