package discord

import (
	"bufio"
	"fmt"
	"io"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
)

type (
	replyWriter struct {
		session *discordgo.Session
		message *discordgo.Message
	}
)

const (
	HelpCommand  commands.Name = "help"
	PermsCommand commands.Name = "perms"
)

func init() {
	commands.Register(HelpCommand, "list all commands", PublicCategory)
	commands.Register(PermsCommand, "show current command permissions", AdminCategory)
}

func (b *Bot) onMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID != b.State.User.ID && len(message.Content) > 1 && b.listenToChannel(Snowflake(message.ChannelID)) {
		b.handleCommandMessage(session, message.Message)
	}
}

func (b *Bot) listenToChannel(channelId Snowflake) bool {
	for _, id := range b.ChannelIDs {
		if id == channelId {
			return true
		}
	}
	return false
}

func (b *Bot) handleCommandMessage(session *discordgo.Session, message *discordgo.Message) {
	replyWriter := replyWriter{session, message}
	writer := bufio.NewWriter(replyWriter)
	actor := &messageActor{message}
	var command *commands.Command
	var err error

	logger := log.WithFields(actor).WithField("message", message.Content)

	defer func() {
		if err != nil {
			_, _ = fmt.Fprintf(writer, "**%s**", err)
			logger.WithError(err).Error("command.rejected")
		}
		_ = writer.Flush()
	}()

	logger.Info("command.received")
	command, err = commands.NewCommandFromString(message.Content, actor)
	if err != nil {
		return
	}
	logger = log.WithFields(command)
	command.Reply = writer

	if b.Permissions.IsAllowed(command.Category, command.Actor) {
		b.dispatcher.Dispatch(command)
	} else {
		err = commands.ErrPermissionDenied
	}

	return
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

func (b *Bot) HandleHelpCommand(cmd *commands.Command) error {
	lineFmt := fmt.Sprintf("%%-%ds - %%s\n", commands.MaxCommandNameLen)
	_, _ = io.WriteString(cmd.Reply, "\n```\n")
	for _, def := range commands.Definitions {
		if b.Permissions.IsAllowed(def.Category, cmd.Actor) {
			_, _ = fmt.Fprintf(cmd.Reply, lineFmt, def.Name, def.Description)
		}
	}
	_, _ = io.WriteString(cmd.Reply, "```")
	return nil
}

func (b *Bot) HandlePermCommand(cmd *commands.Command) error {
	_, _ = io.WriteString(cmd.Reply, "Command permissons:\n")
	for _, def := range commands.Definitions {
		items := make(map[string]bool, 10)
		b.Permissions.Explain(def.Category, commands.Consumer(func(item string) {
			items[item] = true
		}))
		_, _ = fmt.Fprintf(cmd.Reply, "`%s`:", def.Name)
		for item := range items {
			_, _ = fmt.Fprintf(cmd.Reply, " %s", item)
		}
		_, _ = cmd.Reply.WriteString("\n")
	}
	return nil
}
