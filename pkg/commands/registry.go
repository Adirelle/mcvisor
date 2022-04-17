package commands

import (
	"fmt"
	"io"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
)

const (
	HelpCommand Name = "help"
	PermCommand Name = "perms"
)

var (
	commands          = make(map[Name]Definition, 10)
	maxCommandNameLen = 0

	EventHandler = events.MakeHandler(handleCommands)
)

func RegisterPublic(name Name, description string) {
	Register(name, description, permissions.Anyone)
}

func Register(name Name, description string, category permissions.Category) {
	RegisterDefinition(Definition{name, description, category})
}

func RegisterDefinition(def Definition) {
	commands[def.Name] = def
	if l := len(def.Name); l > maxCommandNameLen {
		maxCommandNameLen = l
	}
}

func init() {
	RegisterPublic(HelpCommand, "list all commands")
	Register(PermCommand, "show current command permissions", permissions.AdminCategory)
}

func handleCommands(event events.Event) {
	switch {
	case OnCommand(PermCommand, event, HandlePermCommand),
		OnCommand(HelpCommand, event, HandleHelpCommand):
	}
}

func HandleHelpCommand(cmd *Command) error {
	lineFmt := fmt.Sprintf("%%-%ds - %%s\n", maxCommandNameLen)
	_, _ = io.WriteString(cmd.Reply, "\n```\n")
	for _, def := range commands {
		if def.Allow(cmd.Actor) {
			_, _ = fmt.Fprintf(cmd.Reply, lineFmt, def.Name, def.Description)
		}
	}
	_, _ = io.WriteString(cmd.Reply, "```")
	return nil
}

func HandlePermCommand(cmd *Command) error {
	_, _ = io.WriteString(cmd.Reply, "Command permissons:\n")
	for _, c := range commands {
		_, _ = fmt.Fprintf(cmd.Reply, "`%s`: %s\n", c.Name, c.Permission().DescribePermission())
	}
	return nil
}
