package discord

import (
	"context"
	"fmt"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/utils"
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
)

type (
	Bot struct {
		Token      utils.Secret
		GuildID    Snowflake
		dispatcher events.Dispatcher
		events.HandlerBase
		*discordgo.Session
	}
)

func NewBot(config Config, dispatcher events.Dispatcher) *Bot {
	b := &Bot{
		Token:       config.Token,
		GuildID:     config.GuildID,
		dispatcher:  dispatcher,
		HandlerBase: events.MakeHandlerBase(),
	}
	dispatcher.Add(b)
	return b
}

func (b *Bot) GoString() string {
	return "Discord Bot"
}

func (b *Bot) Serve(ctx context.Context) (err error) {
	err = b.connect()
	if err != nil {
		return fmt.Errorf("could not connect to discord: %w", err)
	}
	defer b.disconnect()

	return events.Serve(b.HandlerBase, b.HandleEvent, ctx)
}

func (b *Bot) HandleEvent(event events.Event) {
	// if notif, ok := event.(Notification); ok && b.Session != nil {
	// 	b.handleNotification(notif)
	// }
}

func (b *Bot) connect() (err error) {
	if b.Session != nil {
		return
	}
	log.Debug("discord.connecting")

	if b.Session, err = discordgo.New("Bot " + b.Token.Reveal()); err == nil {
		b.Identify.Intents = discordgo.IntentsGuildMessages
		b.AddHandler(b.onReady)
		b.AddHandler(b.onMessage)

		err = b.Open()
	}

	if err != nil {
		log.WithError(err).Error("discord.connect")
	}
	return
}

func (b *Bot) onReady(session *discordgo.Session, ready *discordgo.Ready) {
	log.WithField("username", ready.User.Username).Info("discord.ready")
}

func (b *Bot) disconnect() (err error) {
	if b.Session == nil {
		return
	}
	log.Debug("discord.disconnecting")

	err = b.Session.Close()
	b.Session = nil

	if err != nil {
		log.WithError(err).Info("discord.disconnect")
	}
	return
}
