package discord

import (
	"context"
	"fmt"
	"log"

	"github.com/Adirelle/mcvisor/pkg/event"
	"github.com/bwmarrin/discordgo"
)

type (
	Bot struct {
		Config
		event.Dispatcher
		*discordgo.Session
	}
)

func NewBot(config Config, dispatcher event.Dispatcher) *Bot {
	return &Bot{Config: config, Dispatcher: dispatcher}
}

func (b *Bot) GoString() string {
	return "Discord Bot"
}

func (b *Bot) AppID() string {
	if b.Session == nil {
		return ""
	} else {
		return b.State.User.ID
	}
}

func (b *Bot) GuildID() string {
	return string(b.Config.GuildID)
}

func (b *Bot) Serve(ctx context.Context) (err error) {
	if b.Session, err = discordgo.New("Bot " + b.Config.Token.Reveal()); err != nil {
		return fmt.Errorf("could not connect to discord: %w", err)
	}

	b.Identify.Intents = discordgo.IntentsGuildMessages
	b.AddHandler(b.handleMessage)

	if err = b.Open(); err != nil {
		return fmt.Errorf("could not open the session: %w", err)
	}
	defer b.disconnect()

	<-ctx.Done()

	return nil
}

func (b *Bot) HandleEvent(ev event.Event) {
	if n, ok := ev.(Notification); ok {
		b.handleNotification(n)
	}
	if c, ok := ev.(Command); ok {
		switch c.Name {
		case HelpCommand:
			b.handleHelpCommand(c)
		case PermCommand:
			b.handlePermCommand(c)
		default:
		}
	}
}

func (b *Bot) disconnect() {
	if b.Session == nil {
		return
	}

	log.Println("disconnecting from discord")
	err := b.Session.Close()
	if err != nil {
		log.Printf("error disconnecting from discord: %s", err)
	}

	b.Session = nil
	log.Println("disconnected from discord")
}
