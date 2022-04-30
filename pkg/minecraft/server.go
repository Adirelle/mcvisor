package minecraft

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/utils"
	"github.com/apex/log"
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
		console    chan *consoleCommand
		outputs    chan ServerOutput
	}

	Status string

	Target string

	targetSetter struct {
		target Target
		server *Server
	}

	consoleCommand struct {
		command string
		reply   chan<- string
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
	ConsoleCommand  commands.Name = "console"
)

var (
	// Interface check
	_ suture.Service         = (*Server)(nil)
	_ Statuser               = (*Server)(nil)
	_ commands.Handler       = (*targetSetter)(nil)
	_ discord.Notification   = Started
	_ discord.StatusProvider = Started
	_ discord.Notification   = StartTarget

	ConsoleCommandTimeout = 5 * time.Second

	ErrStoppedServer = errors.New("server is stopped")
)

func NewServer(conf *Config, dispatcher *events.Dispatcher) *Server {
	s := &Server{
		Config:     conf,
		target:     StopTarget,
		status:     Stopped,
		targets:    events.MakeHandler[Target](),
		pings:      events.MakeHandler[PingerEvent](),
		console:    events.MakeHandler[*consoleCommand](),
		outputs:    events.MakeHandler[ServerOutput](),
		dispatcher: dispatcher,
	}
	commands.Register(StartCommand, "start the server", discord.ControlCategory, &targetSetter{StartTarget, s})
	commands.Register(StopCommand, "stop the server", discord.ControlCategory, &targetSetter{StopTarget, s})
	commands.Register(RestartCommand, "restart the server", discord.ControlCategory, &targetSetter{RestartTarget, s})
	commands.Register(ShutdownCommand, "stop the server *and* mcvisor", discord.AdminCategory, &targetSetter{ShutdownTarget, s})
	commands.Register(StatusCommand, "show serve status", discord.QueryCategory, commands.HandlerFunc(s.handleStatusCommand))
	commands.Register(ConsoleCommand, "send a console command to the server", discord.ControlCategory, commands.HandlerFunc(s.handleConsoleCommand))
	return s
}

func (s *Server) Serve(ctx context.Context) (err error) {
	defer s.dispatcher.Subscribe(s.targets).Cancel()
	defer s.dispatcher.Subscribe(s.pings).Cancel()
	defer s.dispatcher.Subscribe(s.outputs).Cancel()

	var processDone chan struct{}

	for {
		switch {
		case s.target == RestartTarget && s.status == Stopped:
			s.targets <- StartTarget
		case s.target.MustStart() && !s.status.IsOneOf(Starting, Started, Ready, Unreachable):
			s.setStatus(Starting)
			if s.process == nil {
				s.process, err = newProcess(s.Config, s.dispatcher)
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
		case <-processDone:
			if s.process.Err != nil {
				log.WithError(s.process.Err).Info("server.exited")
			}
			processDone = nil
			s.process = nil
			s.setStatus(Stopped)
		case ping := <-s.pings:
			if ping.IsSuccess() && s.status.IsOneOf(Started, Unreachable) {
				s.setStatus(Ready)
			} else if !ping.IsSuccess() && s.status == Ready {
				s.setStatus(Unreachable)
			}
		case newTarget := <-s.targets:
			s.setTarget(newTarget)
		case cmd := <-s.console:
			s.executeConsoleCommand(cmd.command, cmd.reply)
		case output := <-s.outputs:
			log.WithField("output", output).Debug("server.stdout")
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

func (s *Server) executeConsoleCommand(command string, reply chan<- string) {
	defer close(reply)
	_, err := io.WriteString(s.process.Stdin, command+"\n")
	if err == nil {
		log.WithField("command", command).Info("server.console")
		var output ServerOutput
		if output, err = utils.RecvWithTimeout(s.outputs, time.Second); err == nil {
			reply <- "`" + string(output) + "`"
			return
		}
	}
	log.WithError(err).WithField("command", command).Warn("server.console")
	reply <- err.Error()
}

func (s *Server) handleStatusCommand(cmd *commands.Command) (string, error) {
	return fmt.Sprintf("Server %s", s.status), nil
}

func (s *Server) handleConsoleCommand(cmd *commands.Command) (reply string, err error) {
	if !s.status.IsRunning() {
		err = ErrStoppedServer
		return
	}

	ctx, cleanup := context.WithTimeout(context.Background(), ConsoleCommandTimeout)
	defer cleanup()
	replyC := make(chan string)

	cmdStruct := &consoleCommand{
		command: strings.Join(cmd.Arguments, " "),
		reply:   replyC,
	}
	if err = utils.SendWithContext(s.console, cmdStruct, ctx); err != nil {
		return
	}

	reply, err = utils.RecvWithContext(replyC, ctx)

	return
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
