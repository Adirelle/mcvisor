package minecraft

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/apex/log"
)

type (
	process struct {
		Cmd   *exec.Cmd
		Done  chan struct{}
		Err   error
		Stdin io.Writer
		Stop  func()
	}
)

//go:embed log4j.xml
var log4jFile []byte

func newProcess(c *Config) (p *process, err error) {
	if err = os.WriteFile(c.Server.AbsLog4JConf(), log4jFile, os.FileMode(0o644)); err != nil {
		return
	}

	p = &process{
		Done: make(chan struct{}),
	}

	cmdLine := c.Command()

	var stopCtx context.Context
	stopCtx, p.Stop = context.WithCancel(context.Background())
	p.Cmd = exec.CommandContext(stopCtx, cmdLine[0], cmdLine[1:]...)
	p.Cmd.Dir = c.WorkingDir()
	p.Cmd.Env = c.Env()

	if stdout, err := p.Cmd.StdoutPipe(); err == nil {
		go readLines(stdout, ParseStdout)
	} else {
		return nil, err
	}

	if stderr, err := p.Cmd.StderrPipe(); err == nil {
		go readLines(stderr, LogStderr)
	} else {
		return nil, err
	}

	return
}

func (p *process) Start() error {
	if err := p.Cmd.Start(); err != nil {
		return fmt.Errorf("could not start server: %w", err)
	}

	go p.Wait()
	return nil
}

func (p *process) Wait() {
	defer close(p.Done)
	p.Err = p.Cmd.Wait()
}

func readLines(rd io.Reader, f func(string)) {
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		f(scanner.Text())
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		log.WithError(err).Debug("server.readLines")
	}
}

func ParseStdout(data string) {
	log.WithField("output", data).Info("server.stdout")
}

func LogStderr(data string) {
	log.WithField("output", data).Error("server.stderr")
}
