package discord

import (
	"bufio"
	"fmt"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
)

type (
	replyWriter struct {
		session *discordgo.Session
		message *discordgo.Message
	}
)

func (b *Bot) onMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID != b.State.User.ID && len(message.Content) > 1 {
		go b.handleCommandMessage(session, message.Message)
	}
}

func (b *Bot) handleCommandMessage(session *discordgo.Session, message *discordgo.Message) {
	replyWriter := replyWriter{session, message}
	writer := bufio.NewWriter(replyWriter)

	logger := log.
		WithField("username", message.Author.Username).
		WithField("channelID", message.ChannelID).
		WithField("roles", message.Member.Roles).
		WithField("message", message.Content)

	err := b.execCommand(session, message, writer, logger)
	if err != nil {
		fmt.Fprintf(writer, "**%s**", err)
		_ = writer.Flush()
		logger.WithError(err).Error("command.rejected")
	} else {
		logger.Info("command.dispatched")
	}
}

func (b *Bot) execCommand(session *discordgo.Session, message *discordgo.Message, writer *bufio.Writer, logger *log.Entry) error {
	command, err := commands.Parse(message.Content)
	if err != nil {
		return err
	}
	command.Reply = writer
	command.Actor = messageActor{message}

	if !command.IsAllowed() {
		return permissions.ErrPermissionDenied
	}

	b.DispatchEvent(command)
	return nil
}

func (w replyWriter) Write(data []byte) (int, error) {
	if msg, err := w.session.ChannelMessageSendComplex(
		w.message.ChannelID,
		&discordgo.MessageSend{Content: string(data), Reference: w.message.Reference()},
	); err == nil {
		n := len(msg.Content)
		log.WithField("size", n).Debug("command.reply.sent")
		return len(msg.Content), nil
	} else {
		log.WithError(err).Warn("command.reply.error")
		return 0, err
	}
}
