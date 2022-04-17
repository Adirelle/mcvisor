package discord

import (
	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/Adirelle/mcvisor/pkg/utils"
)

type (
	permissionList []PermissionItem

	Config struct {
		Token         utils.Secret                                 `json:"token" validate:"required"`
		GuildID       Snowflake                                    `json:"serverId" validate:"omitempty"`
		CommandPrefix rune                                         `json:"commandPrefix" validate:"omitempty"`
		Permissions   map[permissions.Category]permissionList      `json:"permissions,omitempty" validate:"omitempty"`
		Notifications map[NotificationCategory]NotificationTargets `json:"notifications,omitempty" validate:"omitempty"`
	}
)

func (c Config) Apply() {
	if c.CommandPrefix != 0 {
		commands.Prefix = c.CommandPrefix
	}
	for cat, list := range c.Permissions {
		perms := make([]permissions.Permission, len(list))
		for i, item := range list {
			perms[i] = item.Permission()
		}
		cat.SetPermission(permissions.AnyOf(perms))
	}
}
