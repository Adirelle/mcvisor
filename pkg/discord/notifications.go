package discord

import (
	"io"
	"strings"

	"github.com/apex/log"
)

type (
	Notifier interface {
		Notify(io.Writer)
	}
)

func (b *Bot) HandleNotifier(notifier Notifier) {
	logger := log.WithField("notifier", notifier)
	builder := strings.Builder{}
	notifier.Notify(&builder)
	message := builder.String()
	logger = logger.WithField("message", message)
	logger.Debug("discord.notification")
	if len(message) == 0 {
		return
	}
	for _, channelID := range b.Notifications {
		loggerC := logger.WithField("channel", channelID)
		if _, err := b.Session.ChannelMessageSend(string(channelID), message); err == nil {
			loggerC.Debug("discord.notification")
		} else {
			loggerC.WithError(err).Warn("discord.notification")
		}
	}
}
