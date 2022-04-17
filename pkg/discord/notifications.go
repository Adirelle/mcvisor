package discord

import "github.com/apex/log"

type (
	Notification interface {
		Category() NotificationCategory
		Message() string
	}

	NotificationCategory string

	NotificationTargets []Snowflake
)

const (
	IgnoredCategory NotificationCategory = ""
	StatusCategory  NotificationCategory = "status"
)

func (b *Bot) handleNotification(n Notification) {
	cat := n.Category()
	if cat == IgnoredCategory {
		return
	}
	targets, found := b.Notifications[cat]
	if !found {
		return
	}
	msg := n.Message()
	logger := log.WithFields(log.Fields{"message": msg, "category": cat})
	logger.Debug("discord.notification.sending")
	for _, target := range targets {
		channelID := string(target)
		_, err := b.ChannelMessageSend(channelID, msg)
		if err != nil {
			logger.WithField("channel", channelID).WithError(err).Error("discord.notification.error")
		}
	}
}
