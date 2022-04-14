package discord

import (
	"log"
)

type (
	Notification interface {
		Category() NotificationCategory
		Message() string
	}

	NotificationCategory string

	NotificationTargets []Secret
)

var (
	IgnoredCategory NotificationCategory = ""
	StatusCategory  NotificationCategory = "status"
)

func (b *Bot) handleNotification(n Notification) {
	cat := n.Category()
	targets, found := b.Notifications[cat]
	if !found {
		return
	}
	msg := n.Message()
	for _, target := range targets {
		channelID := target.Reveal()
		_, err := b.ChannelMessageSend(channelID, msg)
		if err != nil {
			log.Printf("could not notify channel %s: %s", channelID, err)
		}
	}
}
