package discord

import (
	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/apex/log"
	"golang.org/x/exp/slices"
)

type (
	Permissions struct {
		Admin   PermissionList `json:"admin,omitempty"`
		Control PermissionList `json:"control,omitempty"`
		Query   PermissionList `json:"query,omitempty"`
		Public  PermissionList `json:"public,omitempty"`

		asList []PermissionList
	}

	PermissionList []PermissionItem

	PermissionItem struct {
		UserID    *Snowflake `json:"userId,omitempty" validate:"omitempty,required_without_all=Role Channel"`
		RoleID    *Snowflake `json:"roleId,omitempty" validate:"omitempty,required_without_all=User Channel"`
		ChannelID *Snowflake `json:"channelId,omitempty" validate:"omitempty,required_without_all=User Role"`
	}

	actor struct {
		UserID    string
		ChannelID string
		RoleIDs   []string
		*Permissions
	}

	category int
)

var (
	PublicCategory  commands.Permission = category(0)
	QueryCategory   commands.Permission = category(1)
	ControlCategory commands.Permission = category(2)
	AdminCategory   commands.Permission = category(3)

	// Interface checks
	_ commands.Actor = (*actor)(nil)
	_ log.Fielder    = (*actor)(nil)
)

func (p *Permissions) IsAllowed(category category, actor *actor) bool {
	for _, list := range p.AsList()[category:] {
		if list.IsAllowed(actor) {
			return true
		}
	}
	return false
}

func (p *Permissions) AsList() []PermissionList {
	if p.asList == nil {
		p.asList = []PermissionList{p.Public, p.Query, p.Control, p.Admin}
	}
	return p.asList
}

func (l PermissionList) IsAllowed(actor *actor) bool {
	for _, item := range l {
		if item.IsAllowed(actor) {
			return true
		}
	}
	return false
}

func (i PermissionItem) IsAllowed(actor *actor) bool {
	return i.UserID.EqualString(actor.UserID) ||
		i.ChannelID.EqualString(actor.ChannelID) ||
		slices.IndexFunc(actor.RoleIDs, i.RoleID.EqualString) != -1
}

func (a *actor) HasPermission(permission commands.Permission) bool {
	cat, ok := permission.(category)
	return permission == commands.AllowAll || (ok && a.Permissions.IsAllowed(cat, a))
}

func (a *actor) Fields() log.Fields {
	return log.Fields{
		"userId":    a.UserID,
		"channelID": a.ChannelID,
		"roleIDs":   a.RoleIDs,
	}
}
