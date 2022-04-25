package commands

type (
	Permission interface{}

	Actor interface {
		HasPermission(Permission) bool
	}

	all string
)

var AllowAll Permission = all("allow")
