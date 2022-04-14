package discord

import (
	"fmt"
	"log"
	"strings"

	"github.com/Adirelle/mcvisor/pkg/event"
	"github.com/bwmarrin/discordgo"
)

type (
	CommandDef struct {
		Name        string
		Description string
		Permission  PermissionCategory
	}

	CommandReceivedEvent struct {
		event.Time
		Name      string
		Reply     func(string)
		Arguments []string
		PrincipalHolder
	}

	messagePrincipalHolder struct {
		*discordgo.Message
	}
)

var (
	CommandReceivedType = event.Type("CommandReceived")

	HelpCommand = "help"

	commands          = make(map[string]CommandDef)
	maxCommandNameLen = 0
)

func RegisterCommand(cmd CommandDef) {
	commands[cmd.Name] = cmd
	if l := len(cmd.Name); l > maxCommandNameLen {
		maxCommandNameLen = l
	}
}

func init() {
	RegisterCommand(CommandDef{Name: HelpCommand, Description: "list all commands"})
}

func (b *Bot) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == b.State.User.ID || len(m.Message.Content) < 2 || m.Message.Content[0] != byte(b.CommandPrefix) {
		return
	}
	parts := strings.Split(m.Message.Content[1:], " ")
	name := parts[0]

	reply := func(message string) {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Content: message, Reference: m.Reference()})
		if err != nil {
			log.Printf("could not reply to %s: %s", m.Author.Username, err)
		}
	}

	def, found := commands[name]
	if !found {
		reply("**Unknown command**")
		return
	}
	principalHolder := messagePrincipalHolder{m.Message}
	if !b.Permissions.Allow(def.Permission, principalHolder) {
		reply("**Permission denied**")
		return
	}
	b.DispatchEvent(CommandReceivedEvent{event.Now(), def.Name, reply, parts[1:], principalHolder})
}

func (b *Bot) handleUserCommand(cmd CommandReceivedEvent) {
	if cmd.Name != HelpCommand {
		return
	}
	builder := strings.Builder{}
	lineFmt := fmt.Sprintf("%c%%-%ds - %%s\n", b.CommandPrefix, maxCommandNameLen)
	builder.WriteString("\n```\n")
	for _, c := range commands {
		if !b.Permissions.Allow(c.Permission, cmd.PrincipalHolder) {
			continue
		}
		fmt.Fprintf(&builder, lineFmt, c.Name, c.Description)
	}
	builder.WriteString("```")
	cmd.Reply(builder.String())
}

func (CommandReceivedEvent) Type() event.Type {
	return CommandReceivedType
}

func (e CommandReceivedEvent) String() string {
	return fmt.Sprintf("command received: %s (%v)", e.Name, e.Arguments)
}

func (m messagePrincipalHolder) HasUser(userID UserID) bool {
	return m.Author != nil && m.Author.ID == string(userID)
}

func (m messagePrincipalHolder) HasRole(roleID RoleID) bool {
	if m.Member == nil {
		return false
	}
	for _, role := range m.Member.Roles {
		if role == string(roleID) {
			return true
		}
	}
	return false
}

func (m messagePrincipalHolder) HasChannel(channelID ChannelID) bool {
	return m.ChannelID == string(channelID)
}
