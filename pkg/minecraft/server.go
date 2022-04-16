package minecraft

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Adirelle/mcvisor/pkg/events"
)

type (
	Server struct {
		Config
		events.Dispatcher
	}

	ServerStarted  struct{ events.Time }
	ServerStarting struct{ events.Time }
	ServerStopped  struct{ events.Time }
	ServerStopping struct{ events.Time }
	ServerFailure  struct {
		events.Time
		Reason error
	}
)

const ServerStoppingDelay = 10 * time.Second

var (
	ServerStartedType  events.Type = "ServerStarted"
	ServerStartingType events.Type = "ServerStarting"
	ServerStoppedType  events.Type = "ServerStopped"
	ServerStoppingType events.Type = "ServerStopping"
	ServerFailureType  events.Type = "ServerFailure"
)

func NewServer(conf Config, dispatcher events.Dispatcher) *Server {
	return &Server{Config: conf, Dispatcher: dispatcher}
}

func (s *Server) GoString() string {
	return fmt.Sprintf("Minecraft Server (%s)", s.WorkingDir)
}

func (s *Server) Serve(ctx context.Context) error {
	s.DispatchEvent(ServerStarting{events.Now()})
	proc, err := s.StartServer()
	if err != nil {
		s.DispatchEvent(ServerFailure{events.Now(), err})
		return err
	}
	s.DispatchEvent(ServerStarted{events.Now()})
	defer func() {
		s.DispatchEvent(ServerStopped{events.Now()})
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
		s.DispatchEvent(ServerFailure{events.Now(), err})
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
		s.DispatchEvent(ServerStopping{events.Now()})
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
	_, err = writer.WriteString(strconv.Itoa(pid))
	writer.Close()
	return err
}

func (ServerStarting) String() string    { return "server starting" }
func (ServerStarting) Type() events.Type { return ServerStartingType }

func (ServerStarted) String() string    { return "server started" }
func (ServerStarted) Type() events.Type { return ServerStartedType }

func (ServerStopping) String() string    { return "server stopping" }
func (ServerStopping) Type() events.Type { return ServerStoppingType }

func (ServerStopped) String() string    { return "server stopped" }
func (ServerStopped) Type() events.Type { return ServerStoppedType }

func (e ServerFailure) String() string  { return fmt.Sprintf("server failure: %s", e.Reason) }
func (ServerFailure) Type() events.Type { return ServerFailureType }
