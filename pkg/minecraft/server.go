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

	ServerStartedEvent time.Time
	ServerStoppedEvent time.Time
	ServerFailureEvent struct {
		time.Time
		reason error
	}

)

func MakeServer(conf Config, handler event.Handler) Server {
	return Server{Config: conf, Handler: handler}
}

func (s Server) Serve(ctx context.Context) error {

	proc, err := s.StartServer()
	if err != nil {
		s.HandleEvent(serverFailed(err))
		return err
	}
	s.HandleEvent(serverStarted())
	defer func() {
		s.HandleEvent(serverStopped())
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
		s.HandleEvent(serverFailed(err))
	}

	return nil
}

func (s Server) StartServer() (proc *os.Process, err error) {
	cmdLine := s.CmdLine()

	attr := os.ProcAttr{
		Dir: s.WorkingDir,
		Env: s.Env(),
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
	return
	case  <-ctx.Done():
		log.Printf("trying to kill the server, pid: %d", proc.Pid)
		err := proc.Kill()
		if (err != nil) {
			log.Printf("could not kill process #%d: %s", proc.Pid, err)
		}
	}
}

func (s Server) WritePid(pidFile string, pid int) error {
	writer, err := os.OpenFile(pidFile, os.O_CREATE|os.O_TRUNC, os.FileMode(0600))
	if err != nil {
		return err
	}
	writer.WriteString(strconv.Itoa(pid))
	return writer.Close()
}

func serverStarted() ServerStartedEvent {
	return ServerStartedEvent(time.Now())
}

func (e ServerStartedEvent) String() string {
	return fmt.Sprintf("server started at %s", event.FormatTime(time.Time(e)))
}

func serverStopped() ServerStoppedEvent {
	return ServerStoppedEvent(time.Now())
}

func (e ServerStoppedEvent) String() string {
	return fmt.Sprintf("server stopped at %s", event.FormatTime(time.Time(e)))
}

func serverFailed(err error) ServerFailureEvent {
	return ServerFailureEvent{time.Now(), err}
}

func (e ServerFailureEvent) String() string {
	return fmt.Sprintf(
		"server failure at %s: %s",
		event.FormatTime(e.Time),
		e.reason,
	)
}
