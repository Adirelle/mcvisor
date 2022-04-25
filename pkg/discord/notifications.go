package discord

import (
	"github.com/apex/log"
)

type (
	Notification interface {
		DiscordNotification() string
	}
)

func (b *Bot) HandleNotification(notification Notification) {
	message := notification.DiscordNotification()
	if len(message) == 0 {
		return
	}

	logger := log.WithField("notification", notification).WithField("message", message)
	logger.Debug("discord.notification")

	for _, channelID := range b.Notifications {
		loggerC := logger.WithField("channel", channelID)
		if _, err := b.Session.ChannelMessageSend(string(channelID), message); err == nil {
			loggerC.Debug("discord.notification")
		} else {
			loggerC.WithError(err).Warn("discord.notification")
		}
	}
}
