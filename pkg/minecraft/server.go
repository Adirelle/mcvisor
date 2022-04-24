package minecraft

import (
	"context"
	"fmt"
	"io"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/thejerf/suture/v4"
	"golang.org/x/exp/slices"
)

type (
	Server struct {
		*Config
		Status
		Target

		dispatcher *events.Dispatcher
		*process
		targets  chan Target
		commands chan *commands.Command
		pings    chan PingerEvent
	}

	Status string

	Target string
)

const (
	Stopped     Status = "stopped"
	Starting    Status = "starting"
	Started     Status = "started"
	Ready       Status = "ready"
	Unreachable Status = "unreachable"
	Stopping    Status = "stopping"

	StartTarget    Target = "start"
	StopTarget     Target = "stop"
	RestartTarget  Target = "restart"
	ShutdownTarget Target = "shutdown"

	StartCommand    commands.Name = "start"
	StopCommand     commands.Name = "stop"
	RestartCommand  commands.Name = "restart"
	ShutdownCommand commands.Name = "shutdown"
)

var (
	commandTargets = map[commands.Name]Target{
		StartCommand:    StartTarget,
		StopCommand:     StopTarget,
		RestartCommand:  RestartTarget,
		ShutdownCommand: ShutdownTarget,
	}

	// Interface check
	_ suture.Service = (*Server)(nil)
)

func init() {
	commands.Register(StartCommand, "start the server", discord.ControlCategory)
	commands.Register(StopCommand, "stop the server", discord.ControlCategory)
	commands.Register(RestartCommand, "restart the server", discord.ControlCategory)
	commands.Register(ShutdownCommand, "stop the server *and* mcvisor", discord.AdminCategory)
}

func NewServer(conf *Config, dispatcher *events.Dispatcher) *Server {
	return &Server{
		Config:     conf,
		Target:     StartTarget,
		Status:     Stopped,
		targets:    events.MakeHandler[Target](),
		commands:   events.MakeHandler[*commands.Command](),
		pings:      events.MakeHandler[PingerEvent](),
		dispatcher: dispatcher,
	}
}

func (s *Server) Serve(ctx context.Context) (err error) {
	defer s.dispatcher.Subscribe(s.targets).Cancel()
	defer s.dispatcher.Subscribe(s.commands).Cancel()
	defer s.dispatcher.Subscribe(s.pings).Cancel()

	var processDone chan struct{}

	for {
		switch {
		case s.Target == RestartTarget && s.Status == Stopped:
			s.setTarget(StartTarget)
			fallthrough
		case s.Target.MustStart() && !s.Status.IsOneOf(Starting, Started, Ready, Unreachable):
			s.setStatus(Starting)
			if s.process == nil {
				s.process, err = newProcess(s.Config)
				if err != nil {
					return err
				}
			}
			if err = s.Start(); err != nil {
				return err
			}
			processDone = s.process.Done
			s.setStatus(Started)
		case s.Target.MustStop() && !s.Status.IsOneOf(Stopping, Stopped):
			if s.process == nil {
				break
			}
			s.setStatus(Stopping)
			s.process.Stop()
		case s.Target == ShutdownTarget && s.Status == Stopped:
			return suture.ErrTerminateSupervisorTree
		default:
		}

		select {
		case newTarget := <-s.targets:
			s.setTarget(newTarget)
		case ping := <-s.pings:
			if ping.IsSuccess() && s.Status.IsOneOf(Started, Unreachable) {
				s.setStatus(Ready)
			} else if !ping.IsSuccess() && s.Status == Ready {
				s.setStatus(Unreachable)
			}
		case cmd := <-s.commands:
			if newTarget, found := commandTargets[cmd.Name]; found {
				s.setTarget(newTarget)
			}
		case <-processDone:
			if s.Err != nil {
			}
			processDone = nil
			s.process = nil
			s.setStatus(Stopped)
		case <-ctx.Done():
			s.setTarget(ShutdownTarget)
		}
	}
}

func (s *Server) setStatus(status Status) {
	if s.Status == status {
		return
	}
	s.Status = status
	s.dispatcher.Dispatch(status)
}

func (s *Server) setTarget(target Target) {
	if s.Target == target {
		return
	}
	s.Target = target
	s.dispatcher.Dispatch(target)
}

func (t Target) MustStart() bool {
	return t == StartTarget
}

func (t Target) MustStop() bool {
	return t == ShutdownTarget || t == StopTarget || t == RestartTarget
}

func (s Status) IsOneOf(status ...Status) bool {
	return slices.Contains(status, s)
}

func (s Status) Notify(writer io.Writer) {
	_, _ = fmt.Fprintf(writer, "**Server %s**", string(s))
}
