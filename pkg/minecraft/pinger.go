package minecraft

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Adirelle/mcvisor/pkg/discord"
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

	PingSucceededEvent struct {
		event.Time
	}

	PingFailedEvent struct {
		event.Time
		Reason error
	}
)

var (
	PingSucceededType = event.Type("PingSucceeded")
	PingFailedType    = event.Type("PingFailed")

	ErrBothQueryAndStatusDisabled = errors.New("both status and query are disabled")
)

func init() {
	discord.RegisterCommand(discord.CommandDef{
		Name:        "online",
		Description: "list online players",
		Permission:  "query",
	})
}

func MakePinger(conf Config, handler event.Handler) Pinger {
	return Pinger{
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
		p.Ping()
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
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
	if err == nil {
		p.Handler.HandleEvent(PingSucceededEvent{event.Now()})
	} else {
		p.Handler.HandleEvent(PingFailedEvent{event.Now(), err})
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

func (p Pinger) HandleEvent(ev event.Event) {
	if c, isCmd := ev.(discord.ReceivedCommandEvent); isCmd && c.Name == "online" {
		if !p.queryEnabled {
			c.Reply("query is disabled on the server")
			return
		}
		response, err := mcstatusgo.FullQuery(ServerHost, p.queryPort, ConnectionTimeout, ResponseTimeout)
		log.Printf("online result: %#v / %s", response, err)
		if err == nil {
			c.Reply(fmt.Sprintf("```\nOnline players:\n%s\n```", strings.Join(response.Players.PlayerList, "\n")))
		} else {
			c.Reply(fmt.Sprintf("server did not reply: %s", err))
		}
	}
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

func (PingSucceededEvent) Type() event.Type { return PingSucceededType }
func (PingSucceededEvent) String() string   { return "ping succeeded" }

func (PingFailedEvent) Type() event.Type { return PingFailedType }
func (e PingFailedEvent) String() string { return fmt.Sprintf("ping failed: %s", e.Reason) }
