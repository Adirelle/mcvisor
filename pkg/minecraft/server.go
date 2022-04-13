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
		event.Handler
	}

	ServerStartedEvent  time.Time
	ServerStartingEvent time.Time
	ServerStoppedEvent  time.Time
	ServerStoppingEvent time.Time
	ServerFailureEvent  struct {
		time.Time
		Reason error
	}
)

const ServerStoppingDelay = 10 * time.Second

func MakeServer(conf Config, handler event.Handler) Server {
	return Server{Config: conf, Handler: handler}
}

func (s Server) Serve(ctx context.Context) error {
	s.HandleEvent(ServerStartingEvent(time.Now()))
	proc, err := s.StartServer()
	if err != nil {
		s.HandleEvent(ServerFailureEvent{Time: time.Now(), Reason: err})
		return err
	}
	s.HandleEvent(ServerStartedEvent(time.Now()))
	defer func() {
		s.HandleEvent(ServerStoppedEvent(time.Now()))
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
		s.HandleEvent(ServerFailureEvent{Time: time.Now(), Reason: err})
	}

	return nil
}

func (s Server) StartServer() (proc *os.Process, err error) {
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

func (s Server) Wait(ctx context.Context, proc *os.Process) (*os.ProcessState, error) {
	ctl := make(chan struct{})
	defer close(ctl)
	go s.KillOnContextDone(ctx, proc, ctl)
	return proc.Wait()
}

func (s Server) KillOnContextDone(ctx context.Context, proc *os.Process, ctl <-chan struct{}) {
	select {
	case <-ctl:
	case <-ctx.Done():
		s.Handler.HandleEvent(ServerStoppingEvent(time.Now()))
		err := proc.Kill()
		if err != nil {
			log.Printf("could not kill process #%d: %s", proc.Pid, err)
		}
		select {
		case <-ctl:
		case <-time.After(ServerStoppingDelay):
			log.Printf("server still alive after %d", ServerStoppingDelay.String())
		}
	}
}

func (s Server) WritePid(pidFile string, pid int) error {
	writer, err := os.OpenFile(pidFile, os.O_CREATE|os.O_TRUNC, os.FileMode(0o600))
	if err != nil {
		return err
	}
	writer.WriteString(strconv.Itoa(pid))
	return writer.Close()
}

func (e ServerStartingEvent) String() string {
	return fmt.Sprintf("server starting at %s", event.FormatTime(time.Time(e)))
}

func (e ServerStartedEvent) String() string {
	return fmt.Sprintf("server started at %s", event.FormatTime(time.Time(e)))
}

func (e ServerStoppingEvent) String() string {
	return fmt.Sprintf("server stopping at %s", event.FormatTime(time.Time(e)))
}

func (e ServerStoppedEvent) String() string {
	return fmt.Sprintf("server stopped at %s", event.FormatTime(time.Time(e)))
}

func (e ServerFailureEvent) String() string {
	return fmt.Sprintf(
		"server failure at %s: %s",
		event.FormatTime(e.Time),
		e.Reason,
	)
}
