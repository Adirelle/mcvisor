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
		Admin   PermissionList `json:"admin" validate:"required"`
		Control PermissionList `json:"control" validate:"required"`
		Query   PermissionList `json:"query" validate:"required"`
		Public  PermissionList `json:"public,omitempty"`
	}

	PermissionList []PermissionItem

	PermissionItem struct {
		UserID    *Snowflake `json:"userId,omitempty" validate:"omitempty,required_without_all=Role Channel"`
		RoleID    *Snowflake `json:"roleId,omitempty" validate:"omitempty,required_without_all=User Channel"`
		ChannelID *Snowflake `json:"channelId,omitempty" validate:"omitempty,required_without_all=User Role"`
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
	_ commands.Permission = (*Permissions)(nil)
	_ commands.Permission = (*PermissionList)(nil)
	_ commands.Permission = (*PermissionItem)(nil)
	_ Actor               = (*messageActor)(nil)
	_ log.Fielder         = (*messageActor)(nil)
)

func (p *Permissions) IsAllowed(category commands.Category, actor commands.Actor) commands.Decision {
	switch category {
	case PublicCategory:
		if p.Public.IsAllowed(category, actor).IsAllowed() {
			return commands.Allowed
		}
		fallthrough
	case QueryCategory:
		if p.Query.IsAllowed(category, actor).IsAllowed() {
			return commands.Allowed
		}
		fallthrough
	case ControlCategory:
		if p.Control.IsAllowed(category, actor).IsAllowed() {
			return commands.Allowed
		}
		fallthrough
	case AdminCategory:
		if p.Admin.IsAllowed(category, actor).IsAllowed() {
			return commands.Allowed
		}
	}
	return commands.Pass
}

func (p *Permissions) Explain(category commands.Category, consumer commands.Consumer) {
	switch category {
	case PublicCategory:
		p.Public.Explain(category, consumer)
		fallthrough
	case QueryCategory:
		p.Query.Explain(category, consumer)
		fallthrough
	case ControlCategory:
		p.Control.Explain(category, consumer)
		fallthrough
	case AdminCategory:
		p.Admin.Explain(category, consumer)
	}
}

func (l PermissionList) IsAllowed(category commands.Category, actor commands.Actor) (decision commands.Decision) {
	for _, item := range l {
		switch item.IsAllowed(category, actor) {
		case commands.Allowed:
			decision = commands.Allowed
		case commands.Denied:
			return commands.Denied
		}
	}
	return
}

func (l PermissionList) Explain(category commands.Category, consumer commands.Consumer) {
	for _, item := range l {
		item.Explain(category, consumer)
	}
}

func (i PermissionItem) IsAllowed(_ commands.Category, cmdActor commands.Actor) (decision commands.Decision) {
	actor, isActor := cmdActor.(Actor)
	if isActor && ((i.UserID != nil && actor.IsUser(*i.UserID)) ||
		(i.RoleID != nil && actor.HasRole(*i.RoleID)) ||
		(i.ChannelID != nil && actor.InChannel(*i.ChannelID))) {
		return commands.Allowed
	}
	return commands.Pass
}

func (i PermissionItem) Explain(category commands.Category, consumer commands.Consumer) {
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
	if len(parts) == 0 {
		consumer("noone")
	} else {
		consumer(strings.Join(parts, "&"))
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
