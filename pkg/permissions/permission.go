package permissions

import (
	"errors"
	"strings"
)

type (
	Action interface {
		PermissionCategory() Category
	}

	Actor interface {
		DescribeActor() string
	}

	Permission interface {
		Allow(Actor) bool
		DescribePermission() string
	}

	Category string

	AnyOf []Permission

	allowAll int
)

const (
	Anyone Category = ""

	QueryCategory   Category = "query"
	ControlCategory Category = "control"
	AdminCategory   Category = "admin"
)

var (
	ErrPermissionDenied = errors.New("permission denied")

	permissions = make(map[Category]Permission)
)

func (c Category) PermissionCategory() Category {
	return c
}

func (c Category) SetPermission(permission Permission) {
	if c == Anyone {
		return
	}
	permissions[c] = permission
}

func (c Category) Permission() Permission {
	if perm, found := permissions[c]; found {
		return perm
	}
	return allowAll(0)
}

func (c Category) Allow(actor Actor) bool {
	return c.Permission().Allow(actor)
}

func (c Category) DescribePermission() string {
	return c.Permission().DescribePermission()
}

func (a AnyOf) Allow(actor Actor) bool {
	for _, item := range a {
		if item.Allow(actor) {
			return true
		}
	}
	return false
}

func (a AnyOf) DescribePermission() string {
	b := strings.Builder{}
	for i, item := range a {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(item.DescribePermission())
	}
	return b.String()
}

func (allowAll) Allow(Actor) bool           { return true }
func (allowAll) DescribePermission() string { return "anyone" }
