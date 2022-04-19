package discord

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slices"
)

const (
	HelpCommand  commands.Name = "help"
	PermsCommand commands.Name = "perms"
)

func init() {
	commands.Register(HelpCommand, "list all commands", PublicCategory)
	commands.Register(PermsCommand, "show current command permissions", AdminCategory)
}

func (b *Bot) HandleMessage(message *discordgo.Message, ctx context.Context) {
	if message.Author.ID == b.State.User.ID ||
		len(message.Content) < 2 ||
		message.Content[0] != b.CommandPrefix[0] ||
		!slices.Contains(b.ChannelIDs, Snowflake(message.ChannelID)) {
		return
	}

	actor := &messageActor{message}
	logger := log.WithField("actor", actor).WithFields(actor)

	cmdCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	response := make(chan string, 2)
	go func() {
		defer cancel()
		if err := b.CommandReply(response, message, cmdCtx); err == nil {
			logger.Debug("discord.command.replied")
		} else {
			logger.WithError(err).Error("discord.command.error")
		}
		logger.WithError(cmdCtx.Err()).Debug("discord.command.context")
	}()

	cmd, err := commands.ParseCommand(message.Content[1:], actor, response)
	if err != nil {
		logger.WithError(err).Error("discord.command.error")
		return
	}

	logger.WithField("command", cmd).Info("discord.command.received")
	b.dispatcher.Dispatch(cmd)
}

func (b *Bot) CommandReply(response <-chan string, message *discordgo.Message, ctx context.Context) (err error) {
	buffer := bytes.Buffer{}
loop:
	for {
		select {
		case data, open := <-response:
			if !open {
				break loop
			}
			_, err = buffer.Write([]byte(data))
			if err != nil {
				break loop
			}
		case <-ctx.Done():
			break loop
		}
	}
	if buffer.Len() > 0 {
		_, err = b.Session.ChannelMessageSendComplex(
			message.ChannelID,
			&discordgo.MessageSend{Content: buffer.String(), Reference: message.Reference()},
		)
	}
	return
}

func (b *Bot) HandleHelpCommand(cmd *commands.Command) {
	defer close(cmd.Response)
	lineFmt := fmt.Sprintf("%%-%ds - %%s\n", commands.MaxCommandNameLen)
	cmd.Response <- "\n```\n"
	for _, def := range commands.Definitions {
		if commands.IsAllowed(def.Category, cmd.Actor) {
			cmd.Response <- fmt.Sprintf(lineFmt, def.Name, def.Description)
		}
	}
	cmd.Response <- "```"
}

func (b *Bot) HandlePermCommand(cmd *commands.Command) {
	defer close(cmd.Response)
	cmd.Response <- "Command permissons:\n"
	for _, def := range commands.Definitions {

		items := make(map[string]bool, 10)
		commands.Explain(def.Category, func(item string) {
			items[item] = true
		})

		cmd.Response <- fmt.Sprintf("`%s`:", def.Name)
		for item := range items {
			cmd.Response <- fmt.Sprintf(" %s", item)
		}
		cmd.Response <- "\n"
	}
}
