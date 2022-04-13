package minecraft

import (
	"context"
	"fmt"
	"time"

	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/event"
)

type (
	ServerStatus int

	StatusService struct {
		ServerStatus
		LastUpdate time.Time
		event.Handler
	}

	ServerStatusChangedEvent struct {
		time.Time
		Status ServerStatus
	}
)

const (
	ServerStopped ServerStatus = iota
	ServerStarting
	ServerStarted
	ServerReady
	ServerUnreachable
	ServerStopping
)

func init() {
	discord.RegisterCommand(discord.CommandDef{
		Name:        "status",
		Description: "check server status",
		Permission:  "query",
	})
}

func NewStatusService(handler event.Handler) *StatusService {
	return &StatusService{Handler: handler, ServerStatus: ServerStopped}
}

func (s *StatusService) Serve(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s *StatusService) HandleEvent(ev event.Event) {
	newStatus := s.ServerStatus.resolve(ev)
	if newStatus != s.ServerStatus {
		s.ServerStatus = newStatus
		s.LastUpdate = time.Now()
		s.Handler.HandleEvent(ServerStatusChangedEvent{Time: s.LastUpdate, Status: s.ServerStatus})
	}
	if c, ok := ev.(discord.ReceivedCommandEvent); ok && c.CommandDef.Name == "status" {
		c.Reply(fmt.Sprintf("Server is %s", s.ServerStatus))
	}
}

func (s ServerStatus) resolve(ev event.Event) ServerStatus {
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

func (e ServerStatusChangedEvent) String() string {
	return fmt.Sprintf("server status changed to %s at %s", e.Status, event.FormatTime(e.Time))
}
