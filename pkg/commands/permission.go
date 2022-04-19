package commands

import (
	"errors"
)

type (
	Permission interface {
		IsAllowed(Category, Actor) bool
		Explain(Category, func(string))
	}

	Category string

	Actor interface{}

	Decision int

	always bool
)

const (
	DenyAll  = always(false)
	AllowAll = always(true)
)

var (
	ErrPermissionDenied = errors.New("permission denied")

	permissions []Permission

	_ Permission = AllowAll
	_ Permission = DenyAll
)

func PushPermissions(perm Permission) {
	permissions = append(permissions, perm)
}

func (a always) IsAllowed(_ Category, _ Actor) bool {
	return bool(a)
}

func (a always) Explain(_ Category, tell func(string)) {
	if bool(a) {
		tell("anyone")
	} else {
		tell("noone")
	}
}

func IsAllowed(category Category, actor Actor) bool {
	for _, perm := range permissions {
		if perm.IsAllowed(category, actor) {
			return true
		}
	}
	return false
}

func Explain(category Category, tell func(string)) {
	for _, perm := range permissions {
		perm.Explain(category, tell)
	}
}
