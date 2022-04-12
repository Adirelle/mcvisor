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

type (
	Pinger struct {
		propertyPath string
		event.Handler
		*pingerSettings
	}

	pingerSettings struct {
		statusEnabled bool
		statusPort uint16
		queryEnabled bool
		queryPort uint16
	}

	PingSucceededEvent time.Time

	PingFailedEvent struct {
		time.Time
		Reason error
	}
)

var ErrBothQueryAndStatusDisabled = errors.New("both status and query are disabled")

func MakePinger(conf Config, handler event.Handler) Pinger {
	return Pinger{
		propertyPath: conf.ServerPropertiesPath(),
		Handler: handler,
		pingerSettings: new(pingerSettings),
	}
}

func (p Pinger) Serve(ctx context.Context) error {

	if err := p.readSettings(); err != nil {
		return err
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <- ctx.Done():
			return nil
		case <-ticker.C:
			p.Ping()
		}
	}
}

func (p Pinger) Ping() {
	var err error
	now := time.Now()
	if p.queryEnabled {
		err = p.sendQuery()
	} else if p.statusEnabled {
		err = p.getStatus()
	} else {
		err = ErrBothQueryAndStatusDisabled
	}
	if err == nil {
		p.HandleEvent(PingSucceededEvent(now))
	} else {
		p.HandleEvent(PingFailedEvent{now, err})
	}
}

func (p Pinger) sendQuery() (err error) {
	_, err = mcstatusgo.BasicQuery("localhost", p.queryPort, 5 * time.Second, 5 * time.Second)
	return
}

func (p Pinger) getStatus() (err error) {
	_, err = mcstatusgo.Status("localhost", p.statusPort, 5 * time.Second, 5 * time.Second)
	return
}

func (p Pinger) readSettings() error {
	props, err := properties.Load(p.propertyPath)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", p.propertyPath, err)
	}
	p.statusEnabled = props.Bool("enable-status", false)
	p.statusPort = uint16(props.Int("server-port", 25565))
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
