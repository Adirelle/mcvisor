package minecraft

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/apex/log"
)

type (
	Server struct {
		*Config
		dispatcher *events.Dispatcher
	}

	ServerEvent struct {
		*exec.Cmd
		Status
	}
)

const ServerStopTimeout = 10 * time.Second

func NewServer(conf *Config, dispatcher *events.Dispatcher) *Server {
	return &Server{conf, dispatcher}
}

func (s *Server) Serve(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, s.Command(), s.Arguments()...)

	cmd.Dir = s.AbsWorkingDir()
	cmd.Env = s.Env()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = os.Stdout

	startLogger := log.WithField("cmd", cmd.String())
	startLogger.WithField("cmd", cmd.String()).Info("server.starting")

	s.dispatcher.Dispatch(&ServerEvent{cmd, Starting})

	if err := cmd.Start(); err != nil {
		startLogger.WithError(err).Error("server.start.error")
		return err
	}

	runLogger := log.WithField("pid", cmd.Process.Pid)
	runLogger.Info("server.started")

	s.dispatcher.Dispatch(&ServerEvent{cmd, Started})

	err := cmd.Wait()
	if err != nil {
		runLogger.WithError(err).Error("server.stopped")
	} else {
		runLogger.Info("server.stopped")
	}

	s.dispatcher.Dispatch(&ServerEvent{cmd, Stopped})

	return err
}

func (e *ServerEvent) Fields() log.Fields {
	fields := log.Fields{"status": e.Status}
	if e.Process != nil {
		fields["pid"] = e.Process.Pid
	}
	if e.ProcessState != nil {
		fields["exitCode"] = e.ProcessState.ExitCode()
	}
	return fields
}
