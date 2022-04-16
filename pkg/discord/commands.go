package discord

import (
	"bufio"
	"fmt"
	"log"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/permissions"
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
	var err error
	var command commands.Command
	replyWriter := replyWriter{session, message}
	writer := bufio.NewWriter(replyWriter)

	defer func() {
		if err != nil {
			fmt.Fprintf(writer, "**%s**", err)
			log.Printf("command: %s> %s: %s", message.Author.Username, message.Content, err)
		} else {
			log.Printf("command: %s> %s: success", message.Author.Username, message.Content)
		}
	}()

	command, err = commands.Parse(message.Content)
	if err != nil {
		return
	}
	command.Reply = writer
	command.Actor = messageActor{message}

	if command.IsAllowed() {
		b.DispatchEvent(command)
	} else {
		err = permissions.ErrPermissionDenied
	}
}

func (w replyWriter) Write(data []byte) (int, error) {
	if msg, err := w.session.ChannelMessageSendComplex(
		w.message.ChannelID,
		&discordgo.MessageSend{Content: string(data), Reference: w.message.Reference()},
	); err == nil {
		return len(msg.Content), nil
	} else {
		return 0, err
	}
}
