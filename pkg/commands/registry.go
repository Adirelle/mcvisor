package commands

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

type (
	Registry struct {
		definitions       map[Name]*Definition
		maxCommandNameLen int
	}
)

const (
	HelpCommand  Name = "help"
	PermsCommand Name = "perms"
)

var (
	ErrUnknownCommand   = errors.New("unknown command")
	ErrPermissionDenied = errors.New("permission denied")

	builderPool = &sync.Pool{New: func() any { return &strings.Builder{} }}
)

func NewRegistry() *Registry {
	r := &Registry{definitions: make(map[Name]*Definition)}
	r.Register(HelpCommand, "list all commands", AllowAll, HandlerFunc(r.handleHelpCommand))
	return r
}

func (r *Registry) Register(name Name, description string, permission Permission, handler Handler) {
	r.RegisterDefinition(Definition{name, description, permission, handler})
}

func (r *Registry) RegisterDefinition(def Definition) {
	r.definitions[def.Name] = &def
	if l := len(def.Name); l > r.maxCommandNameLen {
		r.maxCommandNameLen = l
	}
}

func (r *Registry) HandleCommand(cmd *Command) (string, error) {
	def, found := r.definitions[cmd.Name]
	switch {
	case !found:
		return "", ErrUnknownCommand
	case !cmd.Actor.HasPermission(def.Permission):
		return "", ErrPermissionDenied
	default:
		return def.HandleCommand(cmd)
	}
}

func (r *Registry) handleHelpCommand(cmd *Command) (string, error) {
	builder := builderPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		builderPool.Put(builder)
	}()

	_, _ = builder.WriteString("\n```\n")
	lineFmt := fmt.Sprintf("%%-%ds - %%s\n", r.maxCommandNameLen)
	for _, def := range r.definitions {
		_, _ = fmt.Fprintf(builder, lineFmt, def.Name, def.Description)
	}
	_, _ = builder.WriteString("```")

	return builder.String(), nil
}
