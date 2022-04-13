package discord

type (
	Secret string

	Config struct {
		Token         Secret                         `json:"token" validate:"required"`
		GuildID       Secret                         `json:"serverId" validate:"omitempty,numeric"`
		Permissions   map[string]PrincipalList       `json:"permissions,omitempty" validate:"omitempty"`
		Notifications map[string]NotificationTargets `json:"notifications,omitempty" validate:"omitempty"`
	}
)

func (c *Config) ConfigureDefaults() {
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
