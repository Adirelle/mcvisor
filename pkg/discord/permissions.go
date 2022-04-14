package discord

import (
	"fmt"
	"io"
	"strings"
)

type (
	UserID    Secret
	ChannelID Secret
	RoleID    Secret

	PrincipalHolder interface {
		HasUser(userID UserID) bool
		HasChannel(channelID ChannelID) bool
		HasRole(roleID RoleID) bool
	}

	Permission interface {
		accept(PrincipalHolder) bool
		Discord() string
	}

	PermissionConfig struct {
		*UserID    `json:"userId" validate:"omitempty,numeric"`
		*RoleID    `json:"roleId" validate:"omitempty,numeric"`
		*ChannelID `json:"channelId" validate:"omitempty,numeric"`
	}

	PermissionList []PermissionConfig

	PermissionCategory string

	PermissionMap map[PermissionCategory]PermissionList
)

const (
	PermCommand = "perms"
)

var (
	QueryPermissionCategory   PermissionCategory = "query"
	ControlPermissionCategory PermissionCategory = "control"
	AdminPermissionCategory   PermissionCategory = "admin"
)

func (c *ChannelID) accept(h PrincipalHolder) bool {
	return c != nil && h.HasChannel(*c)
}

func (c *ChannelID) Discord() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("<#%s>", *c)
}

func (r *RoleID) accept(h PrincipalHolder) bool {
	return r != nil && h.HasRole(*r)
}

func (r *RoleID) Discord() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("<@&%s>", *r)
}

func (u *UserID) accept(h PrincipalHolder) bool {
	return u != nil && h.HasUser(*u)
}

func (u *UserID) Discord() string {
	if u == nil {
		return ""
	}
	return fmt.Sprintf("<@%s>", *u)
}

func (c PermissionConfig) accept(h PrincipalHolder) bool {
	return c.UserID.accept(h) || c.RoleID.accept(h) || c.ChannelID.accept(h)
}

func (c PermissionConfig) Discord() string {
	return c.UserID.Discord() + c.RoleID.Discord() + c.ChannelID.Discord()
}

func (ps PermissionList) accept(h PrincipalHolder) bool {
	if len(ps) == 0 {
		return true
	}
	for _, p := range ps {
		if p.accept(h) {
			return true
		}
	}
	return false
}

func (ps PermissionList) Discord() string {
	if len(ps) == 0 {
		return "anyone"
	}
	b := &strings.Builder{}
	for i, p := range ps {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.Discord())
	}
	return b.String()
}

func (pm PermissionMap) Allow(c PermissionCategory, h PrincipalHolder) bool {
	ps, found := pm[c]
	return !found || ps.accept(h)
}

func init() {
	RegisterCommand(CommandDef{
		Name:        PermCommand,
		Description: "show current command permissions",
		Permission:  AdminPermissionCategory,
	})
}

func (b *Bot) handlePermCommand(cmd Command) {
	io.WriteString(cmd.Reply, "Permissions:\n")
	for _, c := range commands {
		perms := b.Permissions[c.Permission]
		fmt.Fprintf(cmd.Reply, "`%c%s`: %s\n", b.CommandPrefix, c.Name, perms.Discord())
	}
}
