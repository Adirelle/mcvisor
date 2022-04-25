package minecraft

import (
	"context"
	"fmt"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/thejerf/suture/v4"
	"golang.org/x/exp/slices"
)

type (
	Server struct {
		*Config

		dispatcher *events.Dispatcher
		status     Status
		target     Target
		process    *process
		targets    chan Target
		commands   chan *commands.Command
		pings      chan PingerEvent
	}

	Status string

	Target string

	targetSetter struct {
		target Target
		server *Server
	}
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

	StatusCommand   commands.Name = "status"
	StartCommand    commands.Name = "start"
	StopCommand     commands.Name = "stop"
	RestartCommand  commands.Name = "restart"
	ShutdownCommand commands.Name = "shutdown"
)

var (
	// Interface check
	_ suture.Service         = (*Server)(nil)
	_ Statuser               = (*Server)(nil)
	_ commands.Handler       = (*targetSetter)(nil)
	_ discord.Notification   = Started
	_ discord.StatusProvider = Started
	_ discord.Notification   = StartTarget
)

func init() {
}

func NewServer(conf *Config, dispatcher *events.Dispatcher) *Server {
	s := &Server{
		Config:     conf,
		target:     StopTarget,
		status:     Stopped,
		targets:    events.MakeHandler[Target](),
		pings:      events.MakeHandler[PingerEvent](),
		dispatcher: dispatcher,
	}
	commands.Register(StartCommand, "start the server", discord.ControlCategory, &targetSetter{StartTarget, s})
	commands.Register(StopCommand, "stop the server", discord.ControlCategory, &targetSetter{StopTarget, s})
	commands.Register(RestartCommand, "restart the server", discord.ControlCategory, &targetSetter{RestartTarget, s})
	commands.Register(ShutdownCommand, "stop the server *and* mcvisor", discord.AdminCategory, &targetSetter{ShutdownTarget, s})
	commands.Register(StatusCommand, "show serve status", discord.QueryCategory, commands.HandlerFunc(s.handleStatusCommand))
	return s
}

func (s *Server) Serve(ctx context.Context) (err error) {
	defer s.dispatcher.Subscribe(s.targets).Cancel()
	defer s.dispatcher.Subscribe(s.pings).Cancel()

	var processDone chan struct{}

	for {
		switch {
		case s.target == RestartTarget && s.status == Stopped:
			s.targets <- StartTarget
		case s.target.MustStart() && !s.status.IsOneOf(Starting, Started, Ready, Unreachable):
			s.setStatus(Starting)
			if s.process == nil {
				s.process, err = newProcess(s.Config)
				if err != nil {
					return err
				}
			}
			if err = s.process.Start(); err != nil {
				return err
			}
			processDone = s.process.Done
			s.setStatus(Started)
		case s.target.MustStop() && !s.status.IsOneOf(Stopping, Stopped):
			if s.process == nil {
				break
			}
			s.setStatus(Stopping)
			s.process.Stop()
		case s.target == ShutdownTarget && s.status == Stopped:
			return suture.ErrTerminateSupervisorTree
		default:
		}

		select {
		case newTarget := <-s.targets:
			s.setTarget(newTarget)
		case ping := <-s.pings:
			if ping.IsSuccess() && s.status.IsOneOf(Started, Unreachable) {
				s.setStatus(Ready)
			} else if !ping.IsSuccess() && s.status == Ready {
				s.setStatus(Unreachable)
			}
		case <-processDone:
			if s.process.Err != nil {
			}
			processDone = nil
			s.process = nil
			s.setStatus(Stopped)
		case <-ctx.Done():
			s.Shutdown()
		}
	}
}

func (s *Server) Status() Status {
	return s.status
}

func (s *Server) setStatus(status Status) {
	if s.status == status {
		return
	}
	s.status = status
	s.dispatcher.Dispatch(status)
}

func (s *Server) Start() {
	s.targets <- StartTarget
}

func (s *Server) Shutdown() {
	s.targets <- ShutdownTarget
}

func (s *Server) setTarget(target Target) {
	if s.target == target {
		return
	}
	s.target = target
	s.dispatcher.Dispatch(target)
}

func (s *Server) handleStatusCommand(cmd *commands.Command) (string, error) {
	return fmt.Sprintf("Server %s", s.status), nil
}

func (t Target) DiscordNotification() string {
	switch t {
	case RestartTarget:
		return "**Restarting the server**"
	case StartTarget:
		return "**Starting the server**"
	case StopTarget:
		return "**Stopping the server**"
	case ShutdownTarget:
		return "**Shutting down**"
	default:
		return ""
	}
}

func (t Target) MustStart() bool {
	return t == StartTarget
}

func (t Target) MustStop() bool {
	return t == ShutdownTarget || t == StopTarget || t == RestartTarget
}

func (s *targetSetter) HandleCommand(cmd *commands.Command) (string, error) {
	s.server.setTarget(s.target)
	return "", nil
}

func (s Status) IsOneOf(status ...Status) bool {
	return slices.Contains(status, s)
}

func (s Status) IsRunning() bool {
	return s == Started || s == Ready || s == Unreachable
}

func (s Status) DiscordNotification() string {
	switch s {
	case Ready, Unreachable, Stopped:
		return fmt.Sprintf("**Server %s**", string(s))
	default:
		return ""
	}
}

func (s Status) DiscordStatus() string {
	return fmt.Sprintf("Server %s", string(s))
}
