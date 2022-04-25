package commands

var Default = NewRegistry()

func Register(name Name, description string, permission Permission, handler Handler) {
	Default.Register(name, description, permission, handler)
}

func RegisterDefinition(def Definition) {
	Default.RegisterDefinition(def)
}

func HandleCommand(cmd *Command) (string, error) {
	return Default.HandleCommand(cmd)
}

func HandleCommandLine(line string, actor Actor) (string, error) {
	return Default.HandleCommand(NewCommand(line, actor))
}
