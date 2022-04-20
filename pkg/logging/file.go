package logging

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/thejerf/suture/v4"
)

type (
	FileConfig struct {
		Disabled bool      `json:"disabled,omitempty"`
		Path     string    `json:"path"`
		Level    log.Level `json:"level"`

		entries chan *log.Entry
	}
)

const (
	DefaultFilename = "mcvisor.log"
)

var _ factory = (*FileConfig)(nil)

func (f *FileConfig) CreateLogging() (log.Handler, log.Level, suture.Service) {
	return f, f.Level, f
}

func (f *FileConfig) HandleLog(entry *log.Entry) error {
	if entry.Level >= f.Level {
		f.entries <- entry
	}
	return nil
}

func (f *FileConfig) Serve(ctx context.Context) (err error) {
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
