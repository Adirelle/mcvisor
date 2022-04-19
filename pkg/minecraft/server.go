package minecraft

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
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
		Status
	}
)

const ServerStopTimeout = 10 * time.Second

func NewServer(conf *Config, dispatcher *events.Dispatcher) *Server {
	return &Server{conf, dispatcher}
}

func (s *Server) Serve(ctx context.Context) error {
	err := s.GenerateLog4JConf()
	if err != nil {
		return fmt.Errorf("could not generate log4j.conf: %w", err)
	}

	s.dispatcher.Dispatch(&ServerEvent{Starting})

	cmd, err := s.Start()
	if err != nil {
		log.WithField("cmd", cmd.Args).WithError(err).Error("server.start")
		return fmt.Errorf("could not start server: %w", err)
	}

	log.WithField("cmd", cmd.Args).WithField("pid", cmd.Process.Pid).Info("server.started")
	s.dispatcher.Dispatch(&ServerEvent{Started})

	stoppedCtx, stopped := context.WithCancel(context.Background())
	go s.HitMan(cmd.Process, ctx, stoppedCtx)

	err = cmd.Wait()
	stopped()

	log.WithError(err).WithFields(log.Fields{
		"pid":      cmd.ProcessState.Pid,
		"exitCode": cmd.ProcessState.ExitCode(),
	}).Info("server.stopped")

	s.dispatcher.Dispatch(&ServerEvent{Stopped})

	return err
}

func (s *Server) Start() (*exec.Cmd, error) {
	cmdLine := s.Command()

	cmd := exec.Command(cmdLine[0], cmdLine[1:]...)
	cmd.Dir = s.WorkingDir()
	cmd.Env = s.Env()
	cmd.Stdin = os.Stdin

	if stdout, err := cmd.StdoutPipe(); err == nil {
		go s.ParseStdout(stdout)
	} else {
		return cmd, err
	}

	if stderr, err := cmd.StderrPipe(); err == nil {
		go s.LogStderr(stderr)
	} else {
		return cmd, err
	}

	return cmd, cmd.Start()
}

func (s *Server) HitMan(process *os.Process, kill context.Context, stoppedCtx context.Context) {
	select {
	case <-stoppedCtx.Done():
		return
	case <-kill.Done():
	}
	s.dispatcher.Dispatch(&ServerEvent{Stopping})
	err := process.Signal(os.Kill)
	log.WithField("pid", process.Pid).WithError(err).Info("server.stopping")

	stopTimeout, cleanup := context.WithTimeout(stoppedCtx, 5*time.Second)
	defer cleanup()

	<-stopTimeout.Done()
	if stopTimeout.Err() == context.DeadlineExceeded {
		err = process.Signal(syscall.SIGKILL)
		log.WithField("pid", process.Pid).WithError(err).Warn("server.kill")
	}
}

func (s *Server) ParseStdout(stdout io.ReadCloser) {
	reader := bufio.NewReader(stdout)
	buffer := bytes.Buffer{}
	for {
		data, isPrefix, err := reader.ReadLine()
		if err != nil {
			return
		}
		_, _ = buffer.Write(data)
		if !isPrefix {
			var message string
			if err := json.Unmarshal(buffer.Bytes(), &message); err == nil {
				log.WithField("output", message).Info("server.stdout")
			} else {
				log.WithError(err).WithField("data", buffer.String()).Debug("server.stdout")
			}
			buffer.Reset()
		}
	}
}

func (s *Server) LogStderr(stderr io.ReadCloser) {
	reader := bufio.NewReader(stderr)
	buffer := strings.Builder{}
	for {
		data, isPrefix, err := reader.ReadLine()
		if err != nil {
			return
		}
		_, _ = buffer.Write(data)
		if !isPrefix {
			log.WithField("output", buffer.String()).Error("server.stderr")
			buffer.Reset()
		}
	}
}

func (s *Server) GenerateLog4JConf() error {
	// TODO: something better...
	content := `<?xml version="1.0" encoding="UTF-8"?>
<Configuration status="fatal">
	<Appenders>
		<Console name="console" target="SYSTEM_OUT" >
			<PatternLayout
				pattern='"%enc{%m}{JSON}"%n'
				disableAnsi="true"
				noConsoleNoAnsi="true"
				/>
			<Filters>
				<RegexFilter regex="Generating keypair" onMatch="DENY" onMismatch="NEUTRAL"/>
				<RegexFilter regex="Preparing start region for .*" onMatch="DENY" onMismatch="NEUTRAL"/>
			</Filters>
		</Console>

		<Console name="errors" target="SYSTEM_ERR">
			<PatternLayout pattern="[%level] (%c): %msg%n" />
			<ThresholdFilter level="ERROR" onMatch="ACCEPT" onMismatch="DENY"/>
		</Console>

		<RollingFile name="rolling_server_log" fileName="logs/server.log"
				filePattern="logs/server_%d{yyyy-MM-dd}.log">
			<PatternLayout pattern="%d{yyyy-MM-dd HH:mm:ss} [%level] %msg%n" />
			<Policies>
				<TimeBasedTriggeringPolicy />
			</Policies>
		</RollingFile>
	</Appenders>
	<Loggers>
		<Logger name="net.minecraft.server.MinecraftServer" level="info">
			<AppenderRef ref="console" />
		</Logger>
		<Root level="info">
			<AppenderRef ref="rolling_server_log" />
			<AppenderRef ref="errors" />
		</Root>
	</Loggers>
</Configuration>`
	return os.WriteFile(s.Server.AbsLog4JConf(), []byte(content), os.FileMode(0o644))
}
