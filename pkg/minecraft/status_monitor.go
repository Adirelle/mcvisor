package minecraft

import (
	"context"
	"fmt"
	"time"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/apex/log"
)

type (
	Status string

	StatusMonitor struct {
		Status
		When time.Time
		events.Dispatcher
		events.HandlerBase
	}

	StatusChanged struct {
		Old Status
		New Status
	}
)

const (
	Stopped     Status = "stopped"
	Starting    Status = "starting"
	Started     Status = "started"
	Ready       Status = "ready"
	Unreachable Status = "unreachable"
	Stopping    Status = "stopping"

	StatusCommand commands.Name = "status"
)

func init() {
	commands.Register(StatusCommand, "show server status", permissions.QueryCategory)
}

func NewStatusMonitor(dispatcher events.Dispatcher) *StatusMonitor {
	return &StatusMonitor{Dispatcher: dispatcher, Status: Stopped, HandlerBase: events.MakeHandlerBase()}
}

func (s *StatusMonitor) Serve(ctx context.Context) error {
	return events.Serve(s.HandlerBase, s.HandleEvent, ctx)
}

func (s *StatusMonitor) HandleEvent(event events.Event) {
	switch typedEvent := event.(type) {
	case *commands.Command:
		commands.OnCommand(StatusCommand, event, s.handleStatusCommand)
	case ServerEvent:
		s.setStatus(typedEvent.Status)
	case PingSucceeded:
		if s.Status == Started || s.Status == Unreachable {
			s.setStatus(Ready)
		}
	case PingFailed:
		if s.Status == Ready {
			s.setStatus(Unreachable)
		}
	}
}

func (s *StatusMonitor) setStatus(newStatus Status) {
	oldStatus := s.Status
	if newStatus == oldStatus {
		return
	}
	s.Status = newStatus
	s.When = time.Now()
	log.WithField("status", newStatus).Info("server.status")
	s.DispatchEvent(StatusChanged{New: newStatus, Old: oldStatus})
}

func (s *StatusMonitor) handleStatusCommand(cmd *commands.Command) error {
	_, _ = fmt.Fprintf(cmd.Reply, "Server %s <t:%d:R>", s.Status, s.When.Unix())
	return nil
}
