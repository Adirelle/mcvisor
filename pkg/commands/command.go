package commands

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/apex/log"
)

type (
	Name string

	Definition struct {
		Name        Name
		Description string
		Category
	}

	Command struct {
		*Definition
		Actor
		Arguments []string
		Reply     *bufio.Writer
	}

	CommandHandlerFunc func(cmd *Command) error
)

var (
	Prefix rune = '!'

	ErrNoCommandPrefix = errors.New("missing command prefix")
	ErrUnknownCommand  = errors.New("unknown command")

	// interface checks
	_ events.Event = (*Command)(nil)
	_ log.Fielder  = (*Command)(nil)
)

func (n Name) String() string {
	return fmt.Sprintf("%c%s", Prefix, string(n))
}

func OnCommand(name Name, event events.Event, handler CommandHandlerFunc) bool {
	if cmd, ok := event.(*Command); ok && cmd.Name == name {
		defer cmd.Reply.Flush()
		logger := log.WithFields(cmd)
		logger.Debug("command.handle")
		if err := handler(cmd); err == nil {
			logger.Info("command.success")
		} else {
			fmt.Fprintf(cmd.Reply, "**%s**", err)
			logger.WithError(err).Warn("command.error")
		}
		return true
	}
	return false
}

func (c *Command) String() string {
	return strings.Join(append([]string{string(c.Name)}, c.Arguments...), " ")
}

func (c *Command) Fields() log.Fields {
	fields := log.Fields{
		"command": c.Name,
		"args":    c.Arguments,
	}
	if actor, isFielder := c.Actor.(log.Fielder); isFielder {
		for key, value := range actor.Fields() {
			fields[key] = value
		}
	}
	return fields
}

func NewCommandFromString(line string, actor Actor) (*Command, error) {
	if line[0] != byte(Prefix) {
		return nil, ErrNoCommandPrefix
	}

	words := strings.Split(line[1:], " ")
	name := Name(words[0])
	def, found := Definitions[name]
	if !found {
		return nil, ErrUnknownCommand
	}

	return &Command{def, actor, words[1:], nil}, nil
}
