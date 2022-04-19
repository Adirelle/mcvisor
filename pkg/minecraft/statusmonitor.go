package minecraft

import (
	"context"
	"fmt"
	"time"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/apex/log"
)

type (
	Status string

	StatusMonitor struct {
		dispatcher   *events.Dispatcher
		status       Status
		lastUpdate   time.Time
		serverEvents chan *ServerEvent
		pingerEvents chan PingerEvent
		commands     chan *commands.Command
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
	commands.Register(StatusCommand, "show server status", discord.QueryCategory)
}

func NewStatusMonitor(dispatcher *events.Dispatcher) *StatusMonitor {
	return &StatusMonitor{
		dispatcher:   dispatcher,
		status:       Stopped,
		serverEvents: events.MakeHandler[*ServerEvent](),
		pingerEvents: events.MakeHandler[PingerEvent](),
		commands:     events.MakeHandler[*commands.Command](),
	}
}

func (s *StatusMonitor) Serve(ctx context.Context) error {
	defer s.dispatcher.Subscribe(s.serverEvents).Cancel()
	defer s.dispatcher.Subscribe(s.pingerEvents).Cancel()
	defer s.dispatcher.Subscribe(s.commands).Cancel()

	for {
		select {
		case serverEvent := <-s.serverEvents:
			s.setStatus(serverEvent.Status)
		case pingerEvent := <-s.pingerEvents:
			if pingerEvent.IsSuccess() {
				if s.status == Started || s.status == Unreachable {
					s.setStatus(Ready)
				}
			} else if s.status == Ready {
				s.setStatus(Unreachable)
			}
		case cmd := <-s.commands:
			if cmd.Name == StatusCommand {
				cmd.Response <- fmt.Sprintf("Server %s <t:%d:R>", s.status, s.lastUpdate.Unix())
				close(cmd.Response)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *StatusMonitor) setStatus(status Status) {
	if status == s.status {
		return
	}
	s.status = status
	s.lastUpdate = time.Now()
	log.WithField("status", status).Info("server.status")
	s.dispatcher.Dispatch(status)
}
