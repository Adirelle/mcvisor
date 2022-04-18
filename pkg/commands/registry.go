package commands

var (
	Definitions       = make(map[Name]*Definition, 10)
	MaxCommandNameLen = 0
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
