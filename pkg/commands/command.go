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
		permissions.Actor
		Arguments []string
		Reply     *bufio.Writer
	}

	CommandHandlerFunc func(cmd *Command) error
)

var (
	Prefix rune = '!'

	ErrNoCommandPrefix = errors.New("missing command prefix")
	ErrUnknownCommand  = errors.New("unknown command")

	// interface check
	_ events.Event = (*Command)(nil)
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

func (c *Command) IsAllowed() bool {
	return c.Permission().Allow(c.Actor)
}

func (c *Command) Fields() log.Fields {
	return map[string]interface{}{
		"command": c.Name,
		"args":    c.Arguments,
		"actor":   c.Actor.DescribeActor(),
	}
}

func Parse(line string) (*Command, error) {
	if line[0] != byte(Prefix) {
		return nil, ErrNoCommandPrefix
	}

	words := strings.Split(line[1:], " ")
	name := Name(words[0])
	def, found := commands[name]
	if !found {
		return nil, ErrUnknownCommand
	}

	return &Command{&def, nil, words[1:], nil}, nil
}
