package minecraft

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Adirelle/mcvisor/pkg/event"
	properties "github.com/dmotylev/goproperties"
	"github.com/millkhan/mcstatusgo/v2"
)

const (
	ServerHost        = "localhost"
	DefaultServerPort = 25565
	PingPeriod        = 10 * time.Second
	ConnectionTimeout = 5 * time.Second
	ResponseTimeout   = 5 * time.Second
)

type (
	Pinger struct {
		propertyPath string
		event.Handler
		*pingerSettings
	}

	pingerSettings struct {
		statusEnabled bool
		statusPort    uint16
		queryEnabled  bool
		queryPort     uint16
	}

	PingSucceededEvent time.Time

	PingFailedEvent struct {
		time.Time
		Reason error
	}
)

var ErrBothQueryAndStatusDisabled = errors.New("both status and query are disabled")

func NewPinger(conf Config, handler event.Handler) *Pinger {
	return &Pinger{
		propertyPath:   conf.ServerPropertiesPath(),
		Handler:        handler,
		pingerSettings: new(pingerSettings),
	}
}

func (p Pinger) Serve(ctx context.Context) error {
	if err := p.readSettings(); err != nil {
		return err
	}

	ticker := time.NewTicker(PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			p.Ping()
		}
	}
}

func (p Pinger) Ping() {
	var err error
	if p.queryEnabled {
		err = p.sendQuery()
	} else if p.statusEnabled {
		err = p.getStatus()
	} else {
		err = ErrBothQueryAndStatusDisabled
	}
	now := time.Now()
	if err == nil {
		p.HandleEvent(PingSucceededEvent(now))
	} else {
		p.HandleEvent(PingFailedEvent{now, err})
	}
}

func (p Pinger) sendQuery() (err error) {
	_, err = mcstatusgo.BasicQuery(ServerHost, p.queryPort, ConnectionTimeout, ResponseTimeout)
	return
}

func (p Pinger) getStatus() (err error) {
	_, err = mcstatusgo.Status(ServerHost, p.statusPort, ConnectionTimeout, ResponseTimeout)
	return
}

func (p Pinger) readSettings() error {
	props, err := properties.Load(p.propertyPath)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", p.propertyPath, err)
	}
	p.statusEnabled = props.Bool("enable-status", false)
	p.statusPort = uint16(props.Int("server-port", DefaultServerPort))
	p.queryEnabled = props.Bool("enable-query", false)
	p.queryPort = uint16(props.Int("query.port", int64(p.statusPort)))
	return nil
}

func (e PingSucceededEvent) String() string {
	return fmt.Sprintf("ping succeeded at %s", event.FormatTime(time.Time(e)))
}

func (e PingFailedEvent) String() string {
	return fmt.Sprintf("ping failed at %s: %s", event.FormatTime(e.Time), e.Reason)
}
