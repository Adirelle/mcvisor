package commands

import (
	"fmt"
	"io"

	"github.com/Adirelle/mcvisor/pkg/permissions"
)

const (
	HelpCommand Name = "help"
	PermCommand Name = "perms"
)

var (
	commands          = make(map[Name]Definition, 10)
	maxCommandNameLen = 0
)

func Register(def Definition) {
	commands[def.Name] = def
	if l := len(def.Name); l > maxCommandNameLen {
		maxCommandNameLen = l
	}
}

func init() {
	Register(Definition{HelpCommand, "list all commands", permissions.Anyone})
	Register(Definition{PermCommand, "show current command permissions", permissions.AdminCategory})
}

func HandleHelpCommand(cmd Command) {
	lineFmt := fmt.Sprintf("%%-%ds - %%s\n", maxCommandNameLen)
	io.WriteString(cmd.Reply, "\n```\n")
	for _, def := range commands {
		if def.Allow(cmd) {
			fmt.Fprintf(cmd.Reply, lineFmt, def.Name, def.Description)
		}
	}
	io.WriteString(cmd.Reply, "```")
}

func HandlePermCommand(cmd Command) {
	io.WriteString(cmd.Reply, "Command permissons:\n")
	for _, c := range commands {
		fmt.Fprintf(cmd.Reply, "`%s`: %s\n", c.Name, c.Permission().DescribePermission())
	}
}
