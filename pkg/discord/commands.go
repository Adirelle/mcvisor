package discord

import (
	"fmt"
	"log"
	"time"

	"github.com/Adirelle/mcvisor/pkg/event"
	"github.com/bwmarrin/discordgo"
)

type (
	CommandDef struct {
		Name        string
		Description string
		Permission  string
	}

	ReceivedCommandEvent struct {
		time.Time
		CommandDef
		Reply func(string)
	}
)

func (e ReceivedCommandEvent) String() string {
	return fmt.Sprintf("command %s received at %s", e.Name, event.FormatTime(e.Time))
}

var commands = make(map[string]CommandDef)

func RegisterCommand(cmd CommandDef) {
	commands[cmd.Name] = cmd
}

func (b *Bot) registerCommands() error {
	if b.Session == nil {
		return fmt.Errorf("not connected")
	}
	appID := b.AppID()
	guildID := b.GuildID()

	for _, def := range commands {
		var permissions *discordgo.ApplicationCommandPermissionsList = nil
		if def.Permission != "" {
			if permConfig, ok := b.Config.Permissions[def.Permission]; ok {
				permissions = permConfig.toCommandPermissions(appID, guildID)
			} else {
				return fmt.Errorf("unknown permission for command `%s`: %s", def.Name, def.Permission)
			}
		}
		allowedToAll := permissions == nil

		cmd := &discordgo.ApplicationCommand{
			Name:              def.Name,
			Description:       def.Description,
			DefaultPermission: &allowedToAll,
		}
		result, err := b.ApplicationCommandCreate(appID, guildID, cmd)
		if err != nil {
			return fmt.Errorf("could not register command `%s`: %w", def.Name, err)
		}

		if !allowedToAll {
			if err := b.ApplicationCommandPermissionsEdit(appID, guildID, result.ID, permissions); err != nil {
				return fmt.Errorf("could not set permissions of command `%s`: %w", def.Name, err)
			}
		}
	}

	return nil
}

func (b *Bot) unregisterCommands() error {
	if b.Session == nil {
		return fmt.Errorf("not connected")
	}
	appID := b.AppID()
	guildID := b.GuildID()

	cmds, err := b.ApplicationCommands(appID, guildID)
	if err != nil {
		return err
	}
	for _, cmd := range cmds {
		if err := b.ApplicationCommandDelete(appID, guildID, cmd.ID); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bot) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	log.Printf("received command: %#v", data)
	def, known := commands[data.Name]
	if !known {
		log.Printf("received unknown command: %s", data.Name)
		return
	}
	reply := func(message string) {
		s.InteractionRespond(
			i.Interaction,
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: message,
				},
			},
		)
	}
	event := ReceivedCommandEvent{Time: time.Now(), CommandDef: def, Reply: reply}
	b.Handler.HandleEvent(event)
}

func init() {
	RegisterCommand(
		CommandDef{
			Name:        "online",
			Description: "list online player",
			Permission:  "query",
		},
	)
}
