package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/apex/log"
)

type (
	Name string

	Definition struct {
		Name
		Description string
		Category
	}

	Command struct {
		Name
		Arguments []string
		Actor
		Response chan<- string
	}
)

var (
	Definitions       = make(map[Name]*Definition, 10)
	MaxCommandNameLen = 0

	ErrUnknownCommand = errors.New("unknown command")

	// interface checks
	_ log.Fielder = (*Command)(nil)
)

func Register(name Name, description string, category Category) {
	RegisterDefinition(Definition{name, description, category})
}

func RegisterDefinition(def Definition) {
	Definitions[def.Name] = &def
	if l := len(def.Name); l > MaxCommandNameLen {
		MaxCommandNameLen = l
	}
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

func ParseCommand(line string, actor Actor, response chan<- string) (cmd *Command, err error) {
	defer func() {
		if err != nil {
			response <- fmt.Sprintf("**%s**", err)
			close(response)
		}
	}()

	words := strings.Split(line, " ")
	name := Name(words[0])
	def, found := Definitions[name]
	if !found {
		err = ErrUnknownCommand
		return
	}

	cmd = &Command{def.Name, words[1:], actor, response}
	if !IsAllowed(def.Category, actor) {
		err = ErrPermissionDenied
	}

	return
}
