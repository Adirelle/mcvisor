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
	return &Bot{
		Config:     config,
		dispatcher: dispatcher,
		commands:   events.MakeHandler[*commands.Command](),
		messages:   events.MakeHandler[*discordgo.Message](),
	}
}

func (b *Bot) Serve(ctx context.Context) (err error) {
	err = b.connect(ctx)
	if err != nil {
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
			b.HandleMessage(msg)
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

	if b.Session, err = discordgo.New("Bot " + b.Config.Token.Reveal()); err == nil {
		b.Identify.Intents = discordgo.IntentsGuildMessages
		b.AddHandler(b.onReady)
		b.AddHandler(func(_ *discordgo.Session, message *discordgo.MessageCreate) {
			select {
			case b.messages <- message.Message:
			case <-ctx.Done():
			}
		})

		err = b.Open()
	}

	if err != nil {
		log.WithError(err).Error("discord.connect")
	}
	return
}

func (b *Bot) onReady(_ *discordgo.Session, ready *discordgo.Ready) {
	log.WithField("username", ready.User.Username).Info("discord.ready")
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
