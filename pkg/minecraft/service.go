package minecraft

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Service struct {
	Config
}

func (s Service) Serve(ctx context.Context) error {

	cmdLine := s.CmdLine()
	attr := os.ProcAttr{
		Dir: s.WorkingDir,
		Env: s.Env(),
		Files: []*os.File{nil, os.Stdout, os.Stderr},
	}
	proc, err := os.StartProcess(cmdLine[0], cmdLine, &attr)
	if err != nil {
		cmdLine := strings.Join(cmdLine, " ")
		return fmt.Errorf("could not start `%s`: %w", cmdLine, err)
	}

	pidFile, err := os.OpenFile(s.PIDPath, os.O_CREATE|os.O_TRUNC, os.FileMode(0600))
	if err != nil {
		return fmt.Errorf("could not open %s for writing: %w", s.PIDPath, err)
	}
	pidFile.WriteString(strconv.Itoa(proc.Pid))
	err = pidFile.Close()
	defer os.Remove(s.PIDPath)
	if err != nil {
		return fmt.Errorf("could not write to pidfile: %w", err)
	}

	log.Printf("server started, pid: %d", proc.Pid)

	go func() {
		<-ctx.Done()
		log.Printf("trying to kill the server, pid: %d", proc.Pid)
		err := proc.Kill()
		if (err != nil) {
			log.Printf("could not kill process #%d: %s", proc.Pid, err)
		}
	}()

	state, err := proc.Wait()
	if (err != nil) {
		return fmt.Errorf("error while waiting for process #%d to end: %w", proc.Pid, err)
	}
	log.Printf("server stopped, success: %t, exited: %t, exitCode: %d", state.Success(), state.Exited(), state.ExitCode())

	return nil
}
