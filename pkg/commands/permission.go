package commands

import (
	"errors"
)

type (
	Permission interface {
		IsAllowed(Category, Actor) bool
		Explain(Category, Consumer)
	}

	Category string

	Actor interface{}

	Consumer func(string)
)

var ErrPermissionDenied = errors.New("permission denied")
