package discord

import (
	"context"
	"fmt"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slices"
)

func (b *Bot) HandleMessage(message *discordgo.Message, ctx context.Context) {
	if message.Author.ID == b.State.User.ID ||
		len(message.Content) < 2 ||
		message.Content[0] != b.CommandPrefix[0] ||
		!slices.Contains(b.ChannelIDs, Snowflake(message.ChannelID)) {
		return
	}

	actor := &actor{
		UserID:      message.Author.ID,
		ChannelID:   message.ChannelID,
		Permissions: b.Permissions,
	}
	if message.Member != nil {
		actor.RoleIDs = message.Member.Roles
	}
	logger := log.WithField("actor", actor).WithFields(actor)

	go func() {
		reply, err := commands.HandleCommandLine(message.Content[1:], actor)
		if err == nil {
			logger.WithField("reply", reply).Debug("discord.command.reply")
		} else {
			logger.WithError(err).Warn("discord.command.reply")
			reply = fmt.Sprintf("**%s**", err.Error())

		}
		_, _ = b.Session.ChannelMessageSendComplex(
			message.ChannelID,
			&discordgo.MessageSend{Content: reply, Reference: message.Reference()})
	}()
}
