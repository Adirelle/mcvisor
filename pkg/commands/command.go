package commands

import (
	"strings"

	"github.com/apex/log"
)

type (
	Name string

	Definition struct {
		Name
		Description string
		Permission
		Handler
	}

	Command struct {
		Name
		Arguments []string
		Actor
	}

	Handler interface {
		HandleCommand(*Command) (string, error)
	}

	HandlerFunc func(*Command) (string, error)
)

var (
	// interface checks
	_ log.Fielder = (*Command)(nil)
	_ Handler     = (*HandlerFunc)(nil)
)

func NewCommand(line string, actor Actor) *Command {
	words := strings.Split(line, " ")
	return &Command{Name(words[0]), words[1:], actor}
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

func (f HandlerFunc) HandleCommand(cmd *Command) (string, error) {
	return f(cmd)
}
