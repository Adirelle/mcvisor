package discord

import (
	"context"
	"fmt"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
)

type (
	Bot struct {
		Config
		*discordgo.Session
		dispatcher *events.Dispatcher
		commands   chan *commands.Command
		messages   chan *discordgo.Message
	}
)

func NewBot(config Config, dispatcher *events.Dispatcher) *Bot {
	b := &Bot{
		Config:     config,
		dispatcher: dispatcher,
		commands:   events.MakeHandler[*commands.Command](),
		messages:   events.MakeHandler[*discordgo.Message](),
	}
	commands.PushPermissions(b.Permissions)
	return b
}

func (b *Bot) Serve(ctx context.Context) (err error) {
	err = b.connect(ctx)
	if err != nil {
		log.WithError(err).Error("discord.connect")
		return fmt.Errorf("could not connect to Discord: %w", err)
	}
	defer b.disconnect()

	defer b.dispatcher.Subscribe(b.commands).Cancel()

	for {
		select {
		case cmd := <-b.commands:
			switch cmd.Name {
			case PermsCommand:
				b.HandlePermCommand(cmd)
			case HelpCommand:
				b.HandleHelpCommand(cmd)
			}
		case msg := <-b.messages:
			b.HandleMessage(msg, ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *Bot) connect(ctx context.Context) (err error) {
	if b.Session != nil {
		return
	}
	log.Debug("discord.connecting")

	if b.Session, err = discordgo.New("Bot " + b.Config.Token.Reveal()); err != nil {
		return
	}

	b.Identify.Intents = discordgo.IntentsGuildMessages

	readyC := make(chan *discordgo.Ready)
	b.AddHandlerOnce(func(_ *discordgo.Session, ready *discordgo.Ready) {
		select {
		case readyC <- ready:
		case <-ctx.Done():
		}
	})
	b.AddHandler(func(_ *discordgo.Session, message *discordgo.MessageCreate) {
		select {
		case b.messages <- message.Message:
		case <-ctx.Done():
		}
	})

	if err = b.Open(); err != nil {
		return
	}
	log.Debug("discord.bot.connected")

	select {
	case ready := <-readyC:
		err = b.checkGuildMembership(ready)
		if err == nil {
			err = b.checkChannels(ready)
		}
		log.WithField("username", ready.User.Username).Info("discord.bot.ready")
	case <-ctx.Done():
	}

	return
}

func (b *Bot) checkGuildMembership(ready *discordgo.Ready) error {
	found := false
	for _, guild := range ready.Guilds {
		logger := log.WithField("serverId", guild.ID)
		if guild.ID != string(b.GuildID) {
			if err := b.Session.GuildLeave(guild.ID); err == nil {
				logger.Warn("discord.bot.server.leave")
			} else {
				logger.WithError(err).Error("discord.bot.server.leave")
			}
		} else {
			logger.Info("discord.bot.server.member")
			found = true
		}
	}
	if !found {
		log.WithField("serverId", b.GuildID).Error("discord.bot.server.not_member")
	}
	return nil
}

func (b *Bot) checkChannels(ready *discordgo.Ready) error {
	channelIDs := make(map[string]bool)
	for _, id := range b.ChannelIDs {
		channelIDs[string(id)] = false
	}
	b.Permissions.ForEachChannel(func(id Snowflake) {
		channelIDs[string(id)] = false
	})
	channels, err := b.Session.GuildChannels(b.GuildID.String())
	logger := log.WithField("serverId", b.GuildID)
	if err != nil {
		return fmt.Errorf("could not list channels of server: %w", err)
	}
	for _, channel := range channels {
		if _, found := channelIDs[channel.ID]; found {
			logger.WithFields(log.Fields{
				"channelId":   channel.ID,
				"channelName": channel.Name,
				"serverId":    channel.GuildID,
			})
			delete(channelIDs, channel.ID)
			logger.WithField("channelId", channel.ID).Debug("discord.bot.channel.found")
		}
	}
	for id := range channelIDs {
		logger.WithField("channelId", id).Warn("discord.bot.channel.not_found")
	}
	return nil
}

func (b *Bot) disconnect() {
	if b.Session == nil {
		return
	}
	log.Debug("discord.disconnecting")

	err := b.Session.Close()
	b.Session = nil

	if err != nil {
		log.WithError(err).Info("discord.disconnect")
	}
}
