package minecraft

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/apex/log"
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
)

const ServerStopTimeout = 10 * time.Second

var (
	ServerStartedType  events.Type = "server.started"
	ServerStartingType events.Type = "server.starting"
	ServerStoppedType  events.Type = "server.stopped"
	ServerStoppingType events.Type = "server.stopping"
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
		return err
	}
	logger := log.WithField("pid", proc.Pid)
	ctx = log.NewContext(ctx, logger)

	s.DispatchEvent(ServerStarted{events.Now()})
	defer func() {
		s.DispatchEvent(ServerStopped{events.Now()})
	}()

	if err := s.WritePid(s.PidFile, proc.Pid); err == nil {
		defer os.Remove(s.PidFile)
	} else {
		logger.WithError(err).WithField("path", s.PidFile).Warn("server.pidFile.error")
	}

	killCtx, cancelKill := context.WithCancel(ctx)

	go func() {
		select {
		case <-ctx.Done():
			logger.Debug("serker.kill")
			s.DispatchEvent(ServerStopping{events.Now()})
			err := proc.Kill()
			if err != nil {
				logger.WithError(err).Error("server.kill.error")
			}
		case <-killCtx.Done():
		}
	}()

	state, err := proc.Wait()
	cancelKill()

	logger.WithError(err).WithFields(log.Fields{
		"success":  state.Success(),
		"exited":   state.Exited(),
		"exitCode": state.ExitCode(),
	}).Info("server.stopped")

	return err
}

func (s *Server) StartServer() (proc *os.Process, err error) {
	cmdLine := s.CmdLine()
	attr := os.ProcAttr{
		Dir:   s.WorkingDir,
		Env:   s.Env(),
		Files: []*os.File{nil, nil, os.Stderr},
	}

	logger := log.WithField("commandLine", cmdLine)
	logger.Debug("server.start")
	proc, err = os.StartProcess(cmdLine[0], cmdLine, &attr)
	if err == nil {
		logger.Info("server.started")
	} else {
		logger.WithError(err).Error("server.start.error")
	}
	return
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
