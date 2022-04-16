package discord

import (
	"fmt"

	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/Adirelle/mcvisor/pkg/utils"
	"github.com/bwmarrin/discordgo"
)

type (
	messageActor struct{ *discordgo.Message }

	AllowUser    utils.Secret
	AllowRole    utils.Secret
	AllowChannel utils.Secret

	PermissionItem struct {
		*AllowUser    `json:"userId" validate:"omitempty,numeric"`
		*AllowRole    `json:"roleId" validate:"omitempty,numeric"`
		*AllowChannel `json:"channelId" validate:"omitempty,numeric"`
	}
)

func (a messageActor) DescribeActor() string {
	return a.Author.Username
}

func (u AllowUser) Allow(actor permissions.Actor) bool {
	msg, isMessage := actor.(messageActor)
	return isMessage && msg.Author.ID == string(u)
}

func (u AllowUser) DescribePermission() string {
	return fmt.Sprintf("<@%s>", u)
}

func (r AllowRole) Allow(actor permissions.Actor) bool {
	msg, isMessage := actor.(messageActor)
	if !isMessage || msg.Member == nil {
		return false
	}
	for _, role := range msg.Member.Roles {
		if role == string(r) {
			return true
		}
	}
	return false
}

func (r AllowRole) DescribePermission() string {
	return fmt.Sprintf("<@&%s>", r)
}

func (c AllowChannel) Allow(actor permissions.Actor) bool {
	msg, isMessage := actor.(messageActor)
	return isMessage && msg.ChannelID == string(c)
}

func (c AllowChannel) DescribePermission() string {
	return fmt.Sprintf("<#%s>", c)
}

func (i PermissionItem) Permission() permissions.Permission {
	if i.AllowUser != nil {
		return *i.AllowUser
	}
	if i.AllowRole != nil {
		return *i.AllowRole
	}
	if i.AllowChannel != nil {
		return *i.AllowChannel
	}
	panic("empty permission")
}
