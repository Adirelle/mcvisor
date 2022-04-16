package discord

import (
	"context"
	"fmt"
	"log"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/utils"
	"github.com/bwmarrin/discordgo"
)

type (
	Bot struct {
		Token         utils.Secret
		GuildID       Snowflake
		Notifications map[NotificationCategory]NotificationTargets
		events.Dispatcher
		*discordgo.Session
	}

	BotReady         struct{ events.Time }
	BotDisconnecting struct{ events.Time }
)

const (
	BotReadyType         events.Type = "BotReady"
	BotDisconnectingType events.Type = "BotDisconnecting"
)

func NewBot(config Config, dispatcher events.Dispatcher) *Bot {
	return &Bot{
		Token:         config.Token,
		GuildID:       config.GuildID,
		Notifications: config.Notifications,
		Dispatcher:    dispatcher,
	}
}

func (b *Bot) GoString() string {
	return "Discord Bot"
}

func (b *Bot) Serve(ctx context.Context) (err error) {
	if b.Session, err = discordgo.New("Bot " + b.Token.Reveal()); err != nil {
		return fmt.Errorf("could not connect to discord: %w", err)
	}

	b.Identify.Intents = discordgo.IntentsGuildMessages
	b.AddHandler(b.onReady)
	b.AddHandler(b.onMessage)

	if err = b.Open(); err != nil {
		return fmt.Errorf("could not open the session: %w", err)
	}
	defer b.disconnect()

	<-ctx.Done()

	return nil
}

func (b *Bot) HandleEvent(event events.Event) {
	if notif, ok := event.(Notification); ok && b.Session != nil {
		b.handleNotification(notif)
	}
}

func (b *Bot) onReady(session *discordgo.Session, ready *discordgo.Ready) {
	log.Printf("bot ready: connected as %s", ready.User.Username)
	b.DispatchEvent(BotReady{})
}

func (b *Bot) disconnect() {
	if b.Session == nil {
		return
	}

	b.DispatchEvent(BotDisconnecting{})
	err := b.Session.Close()
	if err != nil {
		log.Printf("error disconnecting from discord: %s", err)
	}

	b.Session = nil
}

func (BotReady) String() string                 { return "bot ready" }
func (BotReady) Type() events.Type              { return BotReadyType }
func (BotReady) Category() NotificationCategory { return StatusCategory }
func (BotReady) Message() string                { return "Bot online!" }

func (BotDisconnecting) String() string                 { return "bot disconnecting" }
func (BotDisconnecting) Type() events.Type              { return BotDisconnectingType }
func (BotDisconnecting) Category() NotificationCategory { return StatusCategory }
func (BotDisconnecting) Message() string                { return "Bye bye!" }
