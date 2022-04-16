package minecraft

import (
	"context"
	"fmt"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
)

type (
	ServerStatus int

	StatusMonitor struct {
		ServerStatus
		LastUpdate events.Time
		events.Dispatcher
	}

	ServerStatusChangedEvent struct {
		events.Time
		Status ServerStatus
	}
)

var ServerStatusChangedType = events.Type("ServerStatusChanged")

const (
	ServerStopped ServerStatus = iota
	ServerStarting
	ServerStarted
	ServerReady
	ServerUnreachable
	ServerStopping

	StatusCommand commands.Name = "status"
)

func init() {
	commands.Register(commands.Definition{
		Name:        StatusCommand,
		Description: "check server status",
		Category:    permissions.QueryCategory,
	})
}

func NewStatusMonitor(dispatcher events.Dispatcher) *StatusMonitor {
	return &StatusMonitor{Dispatcher: dispatcher, ServerStatus: ServerStopped}
}

func (s *StatusMonitor) GoString() string {
	return "Status Monitor"
}

func (s *StatusMonitor) Serve(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s *StatusMonitor) HandleEvent(ev events.Event) {
	if c, ok := ev.(commands.Command); ok && c.Name == StatusCommand {
		fmt.Fprintf(c.Reply, "%s since %s", s.ServerStatus, s.LastUpdate.DiscordRelative())
		return
	}

	newStatus := s.ServerStatus.resolve(ev)
	if newStatus != s.ServerStatus {
		s.ServerStatus = newStatus
		s.LastUpdate = events.Now()
		s.DispatchEvent(ServerStatusChangedEvent{events.Time(s.LastUpdate), s.ServerStatus})
	}
}

func (s ServerStatus) resolve(ev events.Event) ServerStatus {
	switch ev.(type) {
	case ServerStartingEvent:
		return ServerStarting
	case ServerStartedEvent:
		return ServerStarted
	case ServerStoppingEvent:
		return ServerStopping
	case ServerStoppedEvent:
		return ServerStopped
	case PingSucceededEvent:
		if s == ServerStarted || s == ServerUnreachable {
			return ServerReady
		}
	case PingFailedEvent:
		if s == ServerReady {
			return ServerUnreachable
		}
	}
	return s
}

func (s ServerStatus) String() string {
	switch s {
	case ServerStopped:
		return "stopped"
	case ServerStarting:
		return "starting"
	case ServerStarted:
		return "started"
	case ServerReady:
		return "ready"
	case ServerUnreachable:
		return "unreachable"
	case ServerStopping:
		return "stopping"
	default:
		return fmt.Sprintf("in an unknown state (%d)", s)
	}
}

func (ServerStatusChangedEvent) Type() events.Type {
	return ServerStatusChangedType
}

func (e ServerStatusChangedEvent) String() string {
	return fmt.Sprintf("status changed to %s", e.Status)
}

func (e ServerStatusChangedEvent) Category() discord.NotificationCategory {
	switch e.Status {
	case ServerStarting, ServerReady, ServerUnreachable, ServerStopped:
		return discord.StatusCategory
	default:
		return discord.IgnoredCategory
	}
}

func (e ServerStatusChangedEvent) Message() string {
	return "Server " + e.Status.String()
}
