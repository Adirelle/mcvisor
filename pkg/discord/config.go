package discord

type (
	Secret string

	Config struct {
		Token         Secret                                       `json:"token" validate:"required"`
		GuildID       Secret                                       `json:"serverId" validate:"omitempty,numeric"`
		CommandPrefix rune                                         `json:"commandPrefix" validate:"omitempty"`
		Permissions   PermissionMap                                `json:"permissions,omitempty" validate:"omitempty"`
		Notifications map[NotificationCategory]NotificationTargets `json:"notifications,omitempty" validate:"omitempty"`
	}
)

func (c *Config) ConfigureDefaults() {
	if c.CommandPrefix == 0 {
		c.CommandPrefix = '!'
	}
}

func (s Secret) Reveal() string {
	return string(s)
}

func (Secret) String() string {
	return "<secret>"
}

func (Secret) GoString() string {
	return "<secret>"
}
