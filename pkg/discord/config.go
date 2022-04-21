package discord

import (
	"github.com/Adirelle/mcvisor/pkg/utils"
)

type (
	Config struct {
		Token         utils.Secret       `json:"token" validate:"required"`
		GuildID       Snowflake          `json:"serverId" validate:"required"`
		ChannelIDs    []Snowflake        `json:"channelIds" validate:"required,min=1"`
		CommandPrefix string             `json:"commandPrefix,omitempty" validate:"required,len=1"`
		Permissions   *Permissions       `json:"permissions" validate:"required"`
		Notifications NotificationConfig `json:"notify,omitempty"`
	}
)

func NewConfig() *Config {
	return &Config{CommandPrefix: "!"}
}
