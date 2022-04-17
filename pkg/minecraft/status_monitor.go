package minecraft

import (
	"context"
	"fmt"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/apex/log"
)

type (
	Status int

	StatusMonitor struct {
		Status
		LastUpdate events.Time
		events.Dispatcher
		events.HandlerBase
	}

	StatusChanged struct {
		events.Time
		Status Status
	}
)

var StatusChangedType = events.Type("server.status.changed")

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
	commands.Register(StatusCommand, "check server status", permissions.QueryCategory)
}

func NewStatusMonitor(dispatcher events.Dispatcher) *StatusMonitor {
	return &StatusMonitor{Dispatcher: dispatcher, Status: Stopped, HandlerBase: events.MakeHandlerBase()}
}

func (s *StatusMonitor) GoString() string {
	return "Status Monitor"
}

func (s *StatusMonitor) Serve(ctx context.Context) error {
	return events.Serve(s.HandlerBase, s.HandleEvent, ctx)
}

func (s *StatusMonitor) HandleEvent(ev events.Event) {
	if c, ok := ev.(commands.Command); ok && c.Name == StatusCommand {
		_, _ = fmt.Fprintf(c.Reply, "%s since %s", s.Status, s.LastUpdate.DiscordRelative())
		_ = c.Reply.Flush()
		return
	}

	newStatus := s.Status.resolve(ev)
	log.WithFields(log.Fields{
		"event":     ev.Type(),
		"oldStatus": s.Status,
		"newStatus": newStatus,
	}).Debug("server.status.iterate")
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
	return StatusChangedType
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
