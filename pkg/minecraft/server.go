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
	err := s.GenerateLog4JConf()
	if err != nil {
		return fmt.Errorf("could not generate log4j.conf: %w", err)
	}

	cmdLine := s.Command()
	cmd := exec.Command(cmdLine[0], cmdLine[1:]...)
	cmd.Dir = s.WorkingDir()
	cmd.Env = s.Env()

	consoleInput, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("could not pipe to stdin: %s", err)
	}

	consoleOutput, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not pipe from stdout: %s", err)
	}

	errorOutput, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not pipe from stderr: %s", err)
	}

	go s.ShutdownOnContext(ctx, cmd, consoleInput)
	go s.ReadStdout(consoleOutput)
	go s.ReadStderr(errorOutput)

	startLogger := log.WithField("cmd", cmd.String())
	startLogger.WithField("cmd", cmd.String()).Info("server.starting")

	s.dispatcher.Dispatch(&ServerEvent{cmd, Starting})

	err = cmd.Start()
	if err != nil {
		startLogger.WithError(err).Error("server.start.error")
		return err
	}

	runLogger := log.WithField("pid", cmd.Process.Pid)
	runLogger.Info("server.started")

	s.dispatcher.Dispatch(&ServerEvent{cmd, Started})

	err = cmd.Wait()
	if err != nil && ctx.Err() != context.Canceled {
		runLogger.WithError(err).Error("server.stopped")
	} else {
		runLogger.Info("server.stopped")
	}

	s.dispatcher.Dispatch(&ServerEvent{cmd, Stopped})

	return err
}

func (s *Server) ShutdownOnContext(ctx context.Context, cmd *exec.Cmd, stdin io.WriteCloser) {
	<-ctx.Done()
	s.dispatcher.Dispatch(&ServerEvent{cmd, Stopping})
	log.WithError(ctx.Err()).Info("server.shutdown.reason")
	_, err := io.WriteString(stdin, `/tellraw @a {"text":"Stopping the server","color":"red"}\n/stop\n`)
	if err != nil {
		log.WithError(err).Error("server.shutdown.gently")
	}
	_ = stdin.Close()
	<-time.After(10 * time.Second)
	log.WithError(ctx.Err()).Warn("server.shutdown.forcefully")
	err = cmd.Process.Kill()
	if err != nil {
		log.WithError(err).Error("server.shutdown.forcefully")
	}
}

func (s *Server) ReadStdout(stdout io.ReadCloser) {
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

func (s *Server) ReadStderr(stderr io.ReadCloser) {
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
