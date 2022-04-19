package commands

import (
	"errors"
	"fmt"
)

type (
	Permission interface {
		IsAllowed(Category, Actor) Decision
		Explain(Category, Consumer)
	}

	Category string

	Actor interface{}

	Consumer func(string)

	allowAll int
	denyAll  int

	Decision int
)

const (
	AllowAll allowAll = 1
	DenyAll  denyAll  = 0

	Pass Decision = iota
	Denied
	Allowed
)

var (
	ErrPermissionDenied = errors.New("permission denied")

	permissions []Permission

	_ Permission   = AllowAll
	_ Permission   = DenyAll
	_ fmt.Stringer = (*Decision)(nil)
)

func PushPermissions(perm Permission) {
	permissions = append(permissions, perm)
}

func (allowAll) IsAllowed(_ Category, _ Actor) Decision {
	return Allowed
}

func (allowAll) Explain(_ Category, consumer Consumer) {
	consumer("anyone")
}

func (denyAll) IsAllowed(_ Category, _ Actor) Decision {
	return Denied
}

func (denyAll) Explain(_ Category, consumer Consumer) {
	consumer("noone")
}

func (c Decision) String() string {
	switch c {
	case Allowed:
		return "allowed"
	case Denied:
		return "denied"
	case Pass:
		return "pass"
	default:
		panic(fmt.Sprintf("invalid Decision value: %d", c))
	}
}

func (c Decision) IsAllowed() bool { return c == Allowed }
func (c Decision) IsDenied() bool  { return c == Denied }
func (c Decision) IsPass() bool    { return c == Pass }

func (c Decision) Combine(o Decision) Decision {
	switch {
	case c.IsDenied() || o.IsDenied():
		return Denied
	case c.IsAllowed() || o.IsAllowed():
		return Allowed
	default:
		return Pass
	}
}

func IsAllowed(category Category, actor Actor) (allowed bool) {
	decision := Pass
	for _, perm := range permissions {
		decision = decision.Combine(perm.IsAllowed(category, actor))
	}
	return decision.IsAllowed()
}

func Explain(category Category, consumer Consumer) {
	for _, perm := range permissions {
		perm.Explain(category, consumer)
	}
}
