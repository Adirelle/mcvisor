package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/level"
)

type (
	Logging struct {
		Console ConsoleLogger `json:"console"`
		File    *FileLogger   `json:"file,omitempty"`
	}

	ConsoleLogger log.Level

	FileLogger struct {
		Disabled bool      `json:"disabled,omitempty"`
		Path     string    `json:"path"`
		Level    log.Level `json:"level"`

		entries chan *log.Entry
	}
)

const (
	DefaultLogFilename = "mcvisor.log"
)

func NewLogging() *Logging {
	return &Logging{
		Console: ConsoleLogger(log.WarnLevel),
		File: &FileLogger{
			Disabled: false,
			Path:     DefaultLogFilename,
			Level:    log.InfoLevel,
			entries:  make(chan *log.Entry, 100),
		},
	}
}

func (l ConsoleLogger) Level() log.Level {
	return log.Level(l)
}

func (l ConsoleLogger) Handler() log.Handler {
	return level.New(cli.Default, l.Level())
}

func (l ConsoleLogger) MarshalJSON() ([]byte, error) {
	return l.Level().MarshalJSON()
}

func (l *ConsoleLogger) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*log.Level)(l))
}

func (f *FileLogger) Handler() log.Handler {
	return log.HandlerFunc(f.handleLog)
}

func (f *FileLogger) handleLog(entry *log.Entry) error {
	if entry.Level >= f.Level {
		f.entries <- entry
	}
	return nil
}

func (f *FileLogger) Serve(ctx context.Context) (err error) {
	var file *os.File
	file, err = os.OpenFile(f.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0o600))
	if err != nil {
		return
	}

	lastSync := time.Now()

	defer func() {
		_ = file.Sync()
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
		if err != nil {
			stdlog.Printf("error logging to %s: %s", f.Path, err)
		}
	}()

	for {
		select {
		case entry := <-f.entries:
			err = f.WriteEntry(file, entry)
			if err != nil {
				return
			}
			if now := time.Now(); now.Sub(lastSync) >= time.Second {
				err = file.Sync()
				if err != nil {
					return
				}
				lastSync = now
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (h *FileLogger) WriteEntry(writer io.Writer, entry *log.Entry) (err error) {
	_, err = fmt.Fprintf(writer, "%s [%s] %s", entry.Timestamp.Format(time.RFC3339), entry.Level, entry.Message)
	if err != nil {
		return
	}

	fields := entry.Fields
	for _, name := range fields.Names() {
		_, err = fmt.Fprintf(writer, " %s=%v", name, fields.Get(name))
		if err != nil {
			return
		}
	}

	_, err = writer.Write([]byte("\n"))

	return
}
