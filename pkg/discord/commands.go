package discord

import (
	"fmt"
	"io"
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

	Command struct {
		*CommandDef
		event.Time
		Arguments []string
		PrincipalHolder
		Reply io.Writer
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
	response := &strings.Builder{}
	go func() {
		defer func() {
			if response.Len() == 0 {
				return
			}
			s.ChannelMessageSendComplex(
				m.ChannelID,
				&discordgo.MessageSend{Content: response.String(), Reference: m.Reference()},
			)
		}()

		def, found := commands[name]
		if !found {
			response.WriteString("**Unknown command**")
			return
		}

		principalHolder := messagePrincipalHolder{m.Message}
		if !b.Permissions.Allow(def.Permission, principalHolder) {
			response.WriteString("**Permission denied**")
			return
		}

		<-b.DispatchEvent(Command{&def, event.Now(), parts[1:], principalHolder, response})
	}()
}

func (b *Bot) handleUserCommand(cmd Command) {
	if cmd.Name != HelpCommand {
		return
	}
	lineFmt := fmt.Sprintf("%c%%-%ds - %%s\n", b.CommandPrefix, maxCommandNameLen)
	io.WriteString(cmd.Reply, "\n```\n")
	for _, c := range commands {
		if !b.Permissions.Allow(c.Permission, cmd.PrincipalHolder) {
			continue
		}
		fmt.Fprintf(cmd.Reply, lineFmt, c.Name, c.Description)
	}
	io.WriteString(cmd.Reply, "```")
}

func (Command) Type() event.Type {
	return CommandReceivedType
}

func (c Command) String() string {
	return fmt.Sprintf("command received: %s (%v)", c.Name, c.Arguments)
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
