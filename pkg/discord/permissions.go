package discord

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

var (
	QueryPermissionCategory   PermissionCategory = "query"
	ControlPermissionCategory PermissionCategory = "control"
)

func (c *ChannelID) accept(h PrincipalHolder) bool {
	return c != nil && h.HasChannel(*c)
}

func (r *RoleID) accept(h PrincipalHolder) bool {
	return r != nil && h.HasRole(*r)
}

func (u *UserID) accept(h PrincipalHolder) bool {
	return u != nil && h.HasUser(*u)
}

func (c PermissionConfig) accept(h PrincipalHolder) bool {
	return c.UserID.accept(h) || c.RoleID.accept(h) || c.ChannelID.accept(h)
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

func (pm PermissionMap) Allow(c PermissionCategory, h PrincipalHolder) bool {
	ps, found := pm[c]
	return !found || ps.accept(h)
}
