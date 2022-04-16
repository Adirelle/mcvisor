package minecraft

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
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
		events.Dispatcher
		*pingerSettings
	}

	pingerSettings struct {
		statusEnabled bool
		statusPort    uint16
		queryEnabled  bool
		queryPort     uint16
	}

	PingSucceeded struct {
		events.Time
	}

	PingFailed struct {
		events.Time
		Reason error
	}
)

var (
	PingSucceededType = events.Type("PingSucceeded")
	PingFailedType    = events.Type("PingFailed")

	ErrBothQueryAndStatusDisabled = errors.New("both status and query are disabled")
)

const (
	OnlineCommand commands.Name = "online"
)

func init() {
	commands.Register(commands.Definition{
		Name:        OnlineCommand,
		Description: "list online players",
		Category:    permissions.QueryCategory,
	})
}

func NewPinger(conf Config, dispatcher events.Dispatcher) *Pinger {
	return &Pinger{
		propertyPath:   conf.ServerPropertiesPath(),
		Dispatcher:     dispatcher,
		pingerSettings: new(pingerSettings),
	}
}

func (*Pinger) GoString() string {
	return "Pinger"
}

func (p *Pinger) Serve(ctx context.Context) error {
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

func (p *Pinger) Ping() {
	var err error
	if p.queryEnabled {
		err = p.sendQuery()
	} else if p.statusEnabled {
		err = p.getStatus()
	} else {
		err = ErrBothQueryAndStatusDisabled
	}
	if err == nil {
		p.DispatchEvent(PingSucceeded{events.Now()})
	} else {
		p.DispatchEvent(PingFailed{events.Now(), err})
	}
}

func (p *Pinger) sendQuery() (err error) {
	_, err = mcstatusgo.BasicQuery(ServerHost, p.queryPort, ConnectionTimeout, ResponseTimeout)
	return
}

func (p *Pinger) getStatus() (err error) {
	_, err = mcstatusgo.Status(ServerHost, p.statusPort, ConnectionTimeout, ResponseTimeout)
	return
}

func (p *Pinger) HandleEvent(ev events.Event) {
	if c, ok := ev.(commands.Command); ok && c.Name == OnlineCommand {
		p.handleOnlineCommand(c)
	}
}

func (p *Pinger) handleOnlineCommand(c commands.Command) {
	if !p.queryEnabled {
		_, _ = io.WriteString(c.Reply, "query is disabled on the server")
		return
	}
	response, err := mcstatusgo.FullQuery(ServerHost, p.queryPort, ConnectionTimeout, ResponseTimeout)
	if err == nil {
		fmt.Fprintf(
			c.Reply,
			"```\nOnline players (%d/%d):\n%s\n```",
			response.Players.Online,
			response.Players.Max,
			strings.Join(response.Players.PlayerList, "\n"),
		)
		return
	}

	log.Printf("could not query server: %s", err)
	if netErr, isNetError := err.(net.Error); isNetError && netErr.Timeout() {
		_, _ = io.WriteString(c.Reply, "could not contact server")
	} else {
		_, _ = io.WriteString(c.Reply, "internal error")
	}
}

func (p *Pinger) readSettings() error {
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

func (PingSucceeded) Type() events.Type { return PingSucceededType }
func (PingSucceeded) String() string    { return "ping succeeded" }

func (PingFailed) Type() events.Type { return PingFailedType }
func (e PingFailed) String() string  { return fmt.Sprintf("ping failed: %s", e.Reason) }
