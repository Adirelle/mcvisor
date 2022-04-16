package discord

import (
	"fmt"

	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/bwmarrin/discordgo"
)

type (
	messageActor struct{ *discordgo.Message }

	AllowedUser    Snowflake
	AllowedRole    Snowflake
	AllowedChannel Snowflake

	PermissionItem struct {
		AllowUser    *Snowflake `json:"userId" validate:"omitempty,required_without_all=AllowRole AllowChannel"`
		AllowRole    *Snowflake `json:"roleId" validate:"omitempty,required_without_all=AllowUser AllowChannel"`
		AllowChannel *Snowflake `json:"channelId" validate:"omitempty,required_without_all=AllowRole AllowUser"`
	}
)

func (a messageActor) DescribeActor() string {
	return a.Author.Username
}

func (u AllowedUser) Allow(actor permissions.Actor) bool {
	msg, isMessage := actor.(messageActor)
	return isMessage && msg.Author.ID == Snowflake(u).String()
}

func (u AllowedUser) DescribePermission() string {
	return fmt.Sprintf("<@%s>", Snowflake(u))
}

func (r AllowedRole) Allow(actor permissions.Actor) bool {
	msg, isMessage := actor.(messageActor)
	if !isMessage || msg.Member == nil {
		return false
	}
	for _, role := range msg.Member.Roles {
		if role == Snowflake(r).String() {
			return true
		}
	}
	return false
}

func (r AllowedRole) DescribePermission() string {
	return fmt.Sprintf("<@&%s>", Snowflake(r))
}

func (c AllowedChannel) Allow(actor permissions.Actor) bool {
	msg, isMessage := actor.(messageActor)
	return isMessage && msg.ChannelID == Snowflake(c).String()
}

func (c AllowedChannel) DescribePermission() string {
	return fmt.Sprintf("<#%s>", Snowflake(c))
}

func (i PermissionItem) Permission() permissions.Permission {
	if i.AllowUser != nil {
		return AllowedUser(*i.AllowUser)
	}
	if i.AllowRole != nil {
		return AllowedRole(*i.AllowRole)
	}
	if i.AllowChannel != nil {
		return AllowedChannel(*i.AllowChannel)
	}
	panic("empty permission")
}
