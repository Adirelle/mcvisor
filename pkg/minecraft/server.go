package minecraft

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Adirelle/mcvisor/pkg/event"
)

type (
	Server struct {
		Config
		event.Dispatcher
	}

	ServerStartedEvent  struct{ event.Time }
	ServerStartingEvent struct{ event.Time }
	ServerStoppedEvent  struct{ event.Time }
	ServerStoppingEvent struct{ event.Time }
	ServerFailureEvent  struct {
		event.Time
		Reason error
	}
)

const ServerStoppingDelay = 10 * time.Second

var (
	ServerStartedType  event.Type = "ServerStarted"
	ServerStartingType event.Type = "ServerStarting"
	ServerStoppedType  event.Type = "ServerStopped"
	ServerStoppingType event.Type = "ServerStopping"
	ServerFailureType  event.Type = "ServerFailure"
)

func NewServer(conf Config, dispatcher event.Dispatcher) *Server {
	return &Server{Config: conf, Dispatcher: dispatcher}
}

func (s *Server) GoString() string {
	return fmt.Sprintf("Minecraft Server (%s)", s.WorkingDir)
}

func (s *Server) Serve(ctx context.Context) error {
	s.DispatchEvent(ServerStartingEvent{event.Now()})
	proc, err := s.StartServer()
	if err != nil {
		s.DispatchEvent(ServerFailureEvent{event.Now(), err})
		return err
	}
	s.DispatchEvent(ServerStartedEvent{event.Now()})
	defer func() {
		s.DispatchEvent(ServerStoppedEvent{event.Now()})
	}()

	if err := s.WritePid(s.PidFile, proc.Pid); err != nil {
		log.Printf("could not write pid into `%s`: %s", s.PidFile, err)
	}

	state, err := s.Wait(ctx, proc)
	if err != nil {
		return fmt.Errorf("error while waiting for process #%d to end: %w", proc.Pid, err)
	}

	if !state.Success() {
		err = fmt.Errorf("exited: %t, exitCode: %d", state.Exited(), state.ExitCode())
		s.DispatchEvent(ServerFailureEvent{event.Now(), err})
	}

	return nil
}

func (s *Server) StartServer() (proc *os.Process, err error) {
	cmdLine := s.CmdLine()

	attr := os.ProcAttr{
		Dir:   s.WorkingDir,
		Env:   s.Env(),
		Files: []*os.File{nil, nil, os.Stderr},
	}

	proc, err = os.StartProcess(cmdLine[0], cmdLine, &attr)
	if err != nil {
		cmdLine := strings.Join(cmdLine, " ")
		err = fmt.Errorf("could not start `%s`: %w", cmdLine, err)
	}

	return
}

func (s *Server) Wait(ctx context.Context, proc *os.Process) (*os.ProcessState, error) {
	ctl := make(chan struct{})
	defer close(ctl)
	go s.KillOnContextDone(ctx, proc, ctl)
	return proc.Wait()
}

func (s *Server) KillOnContextDone(ctx context.Context, proc *os.Process, ctl <-chan struct{}) {
	select {
	case <-ctl:
	case <-ctx.Done():
		s.DispatchEvent(ServerStoppingEvent{event.Now()})
		err := proc.Kill()
		if err != nil {
			log.Printf("could not kill process #%d: %s", proc.Pid, err)
		}
		select {
		case <-ctl:
		case <-time.After(ServerStoppingDelay):
			log.Printf("server still alive after %s", ServerStoppingDelay.String())
		}
	}
}

func (s *Server) WritePid(pidFile string, pid int) error {
	writer, err := os.OpenFile(pidFile, os.O_CREATE|os.O_TRUNC, os.FileMode(0o600))
	if err != nil {
		return err
	}
	writer.WriteString(strconv.Itoa(pid))
	return writer.Close()
}

func (ServerStartingEvent) String() string   { return "server starting" }
func (ServerStartingEvent) Type() event.Type { return ServerStartingType }

func (ServerStartedEvent) String() string   { return "server started" }
func (ServerStartedEvent) Type() event.Type { return ServerStartedType }

func (ServerStoppingEvent) String() string   { return "server stopping" }
func (ServerStoppingEvent) Type() event.Type { return ServerStoppingType }

func (ServerStoppedEvent) String() string   { return "server stopped" }
func (ServerStoppedEvent) Type() event.Type { return ServerStoppedType }

func (e ServerFailureEvent) String() string { return fmt.Sprintf("server failure: %s", e.Reason) }
func (ServerFailureEvent) Type() event.Type { return ServerFailureType }
