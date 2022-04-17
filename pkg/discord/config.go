package discord

import (
	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/Adirelle/mcvisor/pkg/utils"
)

type (
	permissionList []PermissionItem

	Config struct {
		Token         utils.Secret                            `json:"token" validate:"required"`
		GuildID       Snowflake                               `json:"serverId" validate:"omitempty"`
		CommandPrefix string                                  `json:"commandPrefix" validate:"omitempty,len=1"`
		Permissions   map[permissions.Category]permissionList `json:"permissions,omitempty" validate:"omitempty"`
	}
)

func NewConfig() *Config {
	return &Config{
		CommandPrefix: "!",
	}
}

func (c Config) Apply() {
	commands.Prefix = rune(c.CommandPrefix[0])
	for cat, list := range c.Permissions {
		perms := make([]permissions.Permission, len(list))
		for i, item := range list {
			perms[i] = item.Permission()
		}
		cat.SetPermission(permissions.AnyOf(perms))
	}
}
