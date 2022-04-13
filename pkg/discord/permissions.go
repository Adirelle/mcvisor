package discord

import "github.com/bwmarrin/discordgo"

type (
	Principal struct {
		UserID *Secret `json:"userId,omitempty" validate:"omitempty,required_without=RoleID,numeric"`
		RoleID *Secret `json:"roleId,omitempty" validate:"omitempty,required_without=UserID,numeric"`
	}
	PrincipalList []Principal
)

func (p Principal) toCommandPermission() *discordgo.ApplicationCommandPermissions {
	if p.UserID != nil {
		return &discordgo.ApplicationCommandPermissions{
			Type:       discordgo.ApplicationCommandPermissionTypeUser,
			ID:         p.UserID.Reveal(),
			Permission: true,
		}
	}
	if p.RoleID != nil {
		return &discordgo.ApplicationCommandPermissions{
			Type:       discordgo.ApplicationCommandPermissionTypeRole,
			ID:         p.RoleID.Reveal(),
			Permission: true,
		}
	}
	return nil
}

func (ps PrincipalList) toCommandPermissions(appID string, guildID string) *discordgo.ApplicationCommandPermissionsList {
	if len(ps) == 0 {
		return nil
	}
	permissions := make([]*discordgo.ApplicationCommandPermissions, len(ps))
	for i, p := range ps {
		permissions[i] = p.toCommandPermission()
	}
	return &discordgo.ApplicationCommandPermissionsList{Permissions: permissions}
}
