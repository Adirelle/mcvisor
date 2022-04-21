package discord

import (
	"fmt"
	"strings"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slices"
)

type (
	Permissions struct {
		Admin   PermissionList `json:"admin,omitempty"`
		Control PermissionList `json:"control,omitempty"`
		Query   PermissionList `json:"query,omitempty"`
		Public  PermissionList `json:"public,omitempty"`
	}

	PermissionList []PermissionItem

	PermissionItem struct {
		UserID    *Snowflake `json:"userId,omitempty" validate:"omitempty,required_without_all=Role Channel"`
		RoleID    *Snowflake `json:"roleId,omitempty" validate:"omitempty,required_without_all=User Channel"`
		ChannelID *Snowflake `json:"channelId,omitempty" validate:"omitempty,required_without_all=User Role"`
	}

	Permission interface {
		commands.Permission
		ForEachUser(func(Snowflake))
		ForEachRole(func(Snowflake))
		ForEachChannel(func(Snowflake))
	}

	Actor interface {
		commands.Actor
		IsUser(Snowflake) bool
		HasRole(Snowflake) bool
		InChannel(Snowflake) bool
	}

	messageActor struct {
		*discordgo.Message
	}
)

const (
	AdminCategory   commands.Category = "admin"
	ControlCategory commands.Category = "control"
	QueryCategory   commands.Category = "query"
	PublicCategory  commands.Category = "public"
)

var (
	// Interface checks
	_ Permission  = (*Permissions)(nil)
	_ Permission  = (*PermissionList)(nil)
	_ Permission  = (*PermissionItem)(nil)
	_ Actor       = (*messageActor)(nil)
	_ log.Fielder = (*messageActor)(nil)
)

func (p *Permissions) IsAllowed(category commands.Category, actor commands.Actor) bool {
	switch category {
	case PublicCategory:
		if p.Public.IsAllowed(category, actor) {
			return true
		}
		fallthrough
	case QueryCategory:
		if p.Query.IsAllowed(category, actor) {
			return true
		}
		fallthrough
	case ControlCategory:
		if p.Control.IsAllowed(category, actor) {
			return true
		}
		fallthrough
	case AdminCategory:
		if p.Admin.IsAllowed(category, actor) {
			return true
		}
	}
	return false
}

func (p *Permissions) Explain(category commands.Category, tell func(string)) {
	switch category {
	case PublicCategory:
		p.Public.Explain(category, tell)
		fallthrough
	case QueryCategory:
		p.Query.Explain(category, tell)
		fallthrough
	case ControlCategory:
		p.Control.Explain(category, tell)
		fallthrough
	case AdminCategory:
		p.Admin.Explain(category, tell)
	}
}

func (p *Permissions) ForEachUser(visit func(Snowflake)) {
	p.Public.ForEachUser(visit)
	p.Query.ForEachUser(visit)
	p.Control.ForEachUser(visit)
	p.Admin.ForEachUser(visit)
}

func (p *Permissions) ForEachRole(visit func(Snowflake)) {
	p.Public.ForEachRole(visit)
	p.Query.ForEachRole(visit)
	p.Control.ForEachRole(visit)
	p.Admin.ForEachRole(visit)
}

func (p *Permissions) ForEachChannel(visit func(Snowflake)) {
	p.Public.ForEachChannel(visit)
	p.Query.ForEachChannel(visit)
	p.Control.ForEachChannel(visit)
	p.Admin.ForEachChannel(visit)
}

func (l PermissionList) IsAllowed(category commands.Category, actor commands.Actor) bool {
	for _, item := range l {
		if !item.IsAllowed(category, actor) {
			return false
		}
	}
	return true
}

func (l PermissionList) Explain(category commands.Category, tell func(string)) {
	for _, item := range l {
		item.Explain(category, tell)
	}
}

func (l PermissionList) ForEachUser(visit func(Snowflake)) {
	for _, item := range l {
		item.ForEachUser(visit)
	}
}

func (l PermissionList) ForEachRole(visit func(Snowflake)) {
	for _, item := range l {
		item.ForEachRole(visit)
	}
}

func (l PermissionList) ForEachChannel(visit func(Snowflake)) {
	for _, item := range l {
		item.ForEachChannel(visit)
	}
}

func (i PermissionItem) IsAllowed(_ commands.Category, cmdActor commands.Actor) bool {
	actor, isActor := cmdActor.(Actor)
	return isActor && ((i.UserID != nil && actor.IsUser(*i.UserID)) ||
		(i.RoleID != nil && actor.HasRole(*i.RoleID)) ||
		(i.ChannelID != nil && actor.InChannel(*i.ChannelID)))
}

func (i PermissionItem) Explain(category commands.Category, tell func(string)) {
	parts := make([]string, 0, 3)
	if i.UserID != nil {
		parts = append(parts, fmt.Sprintf("<@%s>", *i.UserID))
	}
	if i.RoleID != nil {
		parts = append(parts, fmt.Sprintf("<@&%s>", *i.RoleID))
	}
	if i.ChannelID != nil {
		parts = append(parts, fmt.Sprintf("<#%s>", *i.ChannelID))
	}
	if len(parts) >= 0 {
		tell(strings.Join(parts, "&"))
	}
}

func (i PermissionItem) ForEachUser(visit func(Snowflake)) {
	if i.UserID != nil {
		visit(*i.UserID)
	}
}

func (i PermissionItem) ForEachRole(visit func(Snowflake)) {
	if i.RoleID != nil {
		visit(*i.RoleID)
	}
}

func (i PermissionItem) ForEachChannel(visit func(Snowflake)) {
	if i.ChannelID != nil {
		visit(*i.ChannelID)
	}
}

func (a *messageActor) IsUser(userID Snowflake) bool {
	return a.Author.ID == string(userID)
}

func (a *messageActor) HasRole(roleID Snowflake) bool {
	return a.Member != nil && slices.Contains(a.Member.Roles, string(roleID))
}

func (a *messageActor) InChannel(channelID Snowflake) bool {
	return a.ChannelID == string(channelID)
}

func (a *messageActor) GoString() string {
	return fmt.Sprintf("Message(content=%q, author=%q)", a.Content, a.Author.Username)
}

func (a *messageActor) Fields() log.Fields {
	fields := log.Fields{
		"author":    a.Author.Username,
		"channelID": a.ChannelID,
	}
	if a.Member != nil {
		fields["roleIDs"] = a.Member.Roles
	} else {
		fields["roleIDs"] = nil
	}
	return fields
}
