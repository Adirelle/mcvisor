package logging

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/thejerf/suture/v4"
	"gopkg.in/natefinch/lumberjack.v2"
)

type (
	FileConfig struct {
		Disabled bool      `json:"disabled,omitempty"`
		Level    log.Level `json:"level"`
		*lumberjack.Logger

		entries chan *log.Entry
	}
)

const (
	DefaultFilename = "mcvisor.log"
)

var (
	_ factory        = (*FileConfig)(nil)
	_ suture.Service = (*FileConfig)(nil)
)

func NewFileConfig(baseDir string) *FileConfig {
	return &FileConfig{
		Level: log.InfoLevel,
		Logger: &lumberjack.Logger{
			Filename:   filepath.Join(baseDir, DefaultFilename),
			MaxSize:    10 << 20, // ~
			MaxBackups: 10,
			LocalTime:  true,
			Compress:   true,
		},
		entries: make(chan *log.Entry, 100),
	}
}

func (f *FileConfig) CreateLogging() (log.Handler, log.Level, suture.Service) {
	if f.Disabled {
		return nil, log.FatalLevel, nil
	}
	return f, f.Level, f
}

func (f *FileConfig) HandleLog(entry *log.Entry) error {
	if entry.Level >= f.Level {
		f.entries <- entry
	}
	return nil
}

func (f *FileConfig) Serve(ctx context.Context) (err error) {
	defer func() {
		cerr := f.Logger.Close()
		if err == nil {
			err = cerr
		}
		if err != nil {
			stdlog.Printf("error logging to %s: %s", f.Filename, err)
		}
	}()
	log.WithField("path", f.Filename).Debug("logging.file.started")

	for {
		select {
		case entry := <-f.entries:
			err = f.WriteEntry(f.Logger, entry)
			if err != nil {
				return
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (f *FileConfig) WriteEntry(writer io.Writer, entry *log.Entry) (err error) {
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
