package discord

import (
	"fmt"
	"log"
	"strings"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID != b.State.User.ID && len(message.Content) > 1 {
		go b.handleCommandMessage(session, message.Message)
	}
}

func (b *Bot) handleCommandMessage(session *discordgo.Session, message *discordgo.Message) {
	var err error
	var command commands.Command
	response := &strings.Builder{}

	defer func() {
		if err != nil {
			fmt.Fprintf(response, "**%s**", err)
			log.Printf("command: %s> %s: %s", message.Author.Username, message.Content, err)
		} else {
			log.Printf("command: %s> %s: success", message.Author.Username, message.Content)
		}
		if response.Len() > 0 {
			session.ChannelMessageSendComplex(
				message.ChannelID,
				&discordgo.MessageSend{Content: response.String(), Reference: message.Reference()},
			)
		}
	}()

	command, err = commands.Parse(message.Content)
	if err != nil {
		return
	}
	command.Reply = response
	command.Actor = messageActor{message}

	if command.IsAllowed() {
		<-b.DispatchEvent(command)
	} else {
		err = permissions.ErrPermissionDenied
	}
}
