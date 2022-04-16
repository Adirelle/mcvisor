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
	Status int

	StatusMonitor struct {
		Status
		LastUpdate events.Time
		events.Dispatcher
	}

	StatusChanged struct {
		events.Time
		Status Status
	}
)

var ServerStatusChangedType = events.Type("StatusChanged")

const (
	Stopped Status = iota
	Starting
	Started
	Ready
	Unreachable
	Stopping

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
	return &StatusMonitor{Dispatcher: dispatcher, Status: Stopped}
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
		fmt.Fprintf(c.Reply, "%s since %s", s.Status, s.LastUpdate.DiscordRelative())
		return
	}

	newStatus := s.Status.resolve(ev)
	if newStatus != s.Status {
		s.Status = newStatus
		s.LastUpdate = events.Now()
		s.DispatchEvent(StatusChanged{events.Time(s.LastUpdate), s.Status})
	}
}

func (s Status) resolve(ev events.Event) Status {
	switch ev.(type) {
	case ServerStarting:
		return Starting
	case ServerStarted:
		return Started
	case ServerStopping:
		return Stopping
	case ServerStopped:
		return Stopped
	case PingSucceeded:
		if s == Started || s == Unreachable {
			return Ready
		}
	case PingFailed:
		if s == Ready {
			return Unreachable
		}
	}
	return s
}

func (s Status) String() string {
	switch s {
	case Stopped:
		return "stopped"
	case Starting:
		return "starting"
	case Started:
		return "started"
	case Ready:
		return "ready"
	case Unreachable:
		return "unreachable"
	case Stopping:
		return "stopping"
	default:
		return fmt.Sprintf("in an unknown state (%d)", s)
	}
}

func (StatusChanged) Type() events.Type {
	return ServerStatusChangedType
}

func (e StatusChanged) String() string {
	return fmt.Sprintf("status changed to %s", e.Status)
}

func (e StatusChanged) Category() discord.NotificationCategory {
	switch e.Status {
	case Starting, Ready, Unreachable, Stopped:
		return discord.StatusCategory
	default:
		return discord.IgnoredCategory
	}
}

func (e StatusChanged) Message() string {
	return "Server " + e.Status.String()
}
