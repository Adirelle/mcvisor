package discord

import (
	"time"

	"github.com/apex/log"
)

type (
	StatusProvider interface {
		DiscordStatus() string
	}
)

func (b *Bot) HandleStatusProvider(provider StatusProvider) {
	err := b.Session.UpdateGameStatus(int(time.Now().UnixMilli()), provider.DiscordStatus())
	if err != nil {
		log.WithError(err).Warn("discord.status.update")
	}
}
