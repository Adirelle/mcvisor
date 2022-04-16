package discord

import (
	"log"

	"github.com/Adirelle/mcvisor/pkg/utils"
)

type (
	Notification interface {
		Category() NotificationCategory
		Message() string
	}

	NotificationCategory string

	NotificationTargets []utils.Secret
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
	for _, target := range targets {
		channelID := string(target)
		_, err := b.ChannelMessageSend(channelID, msg)
		if err != nil {
			log.Printf("could not notify channel %s: %s", channelID, err)
		}
	}
}
