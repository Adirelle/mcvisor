package discord

import (
	"log"

	"github.com/Adirelle/mcvisor/pkg/event"
)

type (
	Notification interface {
		Category() string
		Message() string
	}

	NotificationTargets []Secret
)

func (b *Bot) HandleEvent(ev event.Event) {
	notif, isNotif := ev.(Notification)
	if !isNotif {
		return
	}
	cat := notif.Category()
	targets, found := b.Notifications[cat]
	if !found {
		return
	}
	msg := notif.Message()
	for _, target := range targets {
		channelID := target.Reveal()
		_, err := b.ChannelMessageSend(channelID, msg)
		if err != nil {
			log.Printf("could not notify channel %s: %s", channelID, err)
		}
	}
}
