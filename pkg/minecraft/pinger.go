package minecraft

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/apex/log"
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
		events.HandlerBase
	}

	pingerSettings struct {
		statusEnabled bool
		statusPort    uint16
		queryEnabled  bool
		queryPort     uint16
	}

	PingEvent interface {
		events.Event
		isPingEvent()
	}

	PingSucceeded struct {
		events.Time
		Latency time.Duration
	}

	PingFailed struct {
		events.Time
		Reason error
	}
)

var (
	PingSucceededType = events.Type("ping.succeeded")
	PingFailedType    = events.Type("ping.failed")

	ErrBothQueryAndStatusDisabled = errors.New("both status and query are disabled by server configuration")
	ErrQueryDisabled              = errors.New("query is disabled by server configuration")
	ErrQueryTimeout               = errors.New("server did not respond")

	PingDisabled = MakePingFailed(ErrBothQueryAndStatusDisabled)
)

const (
	OnlineCommand commands.Name = "online"
)

func init() {
	commands.Register(OnlineCommand, "list online players", permissions.QueryCategory)
}

func NewPinger(conf Config, dispatcher events.Dispatcher) *Pinger {
	return &Pinger{
		propertyPath:   conf.ServerPropertiesPath(),
		Dispatcher:     dispatcher,
		pingerSettings: new(pingerSettings),
		HandlerBase:    events.MakeHandlerBase(),
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
		select {
		case <-ctx.Done():
			return nil
		case ev := <-p.HandlerBase:
			p.HandleEvent(ev)
		case <-ticker.C:
			p.Ping()
		}
	}
}

func (p *Pinger) Ping() {
	var result PingEvent
	if p.queryEnabled {
		result = p.sendQuery()
	} else if p.statusEnabled {
		result = p.getStatus()
	} else {
		result = PingDisabled
	}
	p.DispatchEvent(result)
}

func (p *Pinger) sendQuery() PingEvent {
	if response, err := mcstatusgo.BasicQuery(ServerHost, p.queryPort, ConnectionTimeout, ResponseTimeout); err == nil {
		return PingSucceeded{events.Now(), response.Latency}
	} else {
		return MakePingFailed(err)
	}
}

func (p *Pinger) getStatus() PingEvent {
	if status, err := mcstatusgo.Status(ServerHost, p.statusPort, ConnectionTimeout, ResponseTimeout); err == nil {
		return PingSucceeded{events.Now(), status.Latency}
	} else {
		return MakePingFailed(err)
	}
}

func (p *Pinger) HandleEvent(event events.Event) {
	switch ev := event.(type) {
	case commands.Command:
		if ev.Name == OnlineCommand {
			p.handleOnlineCommand(ev)
		}
	}
}

func (p *Pinger) handleOnlineCommand(c commands.Command) {
	defer c.Reply.Flush()
	err := p.doOnlineQuery(c.Reply)
	if err != nil {
		_, _ = io.WriteString(c.Reply, "**server unreachable**")
		log.WithError(err).Error("pinger.online")
	}
}

func (p *Pinger) doOnlineQuery(writer io.Writer) error {
	if !p.queryEnabled {
		return ErrQueryDisabled
	}
	if response, err := mcstatusgo.FullQuery(ServerHost, p.queryPort, ConnectionTimeout, ResponseTimeout); err == nil {
		_, _ = fmt.Fprintf(
			writer,
			"```\nOnline players (%d/%d):\n%s\n```",
			response.Players.Online,
			response.Players.Max,
			strings.Join(response.Players.PlayerList, "\n"),
		)
		log.WithFields(log.Fields{
			"latency":     response.Latency,
			"player.list": response.Players.PlayerList,
			"player.max":  response.Players.Max,
		}).Info("pinger.online")
		return nil
	} else {
		return err
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

func (PingSucceeded) isPingEvent()         {}
func (PingSucceeded) Type() events.Type    { return PingSucceededType }
func (PingSucceeded) String() string       { return "ping succeeded" }
func (s PingSucceeded) Fields() log.Fields { return map[string]interface{}{"latency": s.Latency} }

func MakePingFailed(err error) PingFailed {
	return PingFailed{events.Now(), err}
}

func (PingFailed) isPingEvent()         {}
func (PingFailed) Type() events.Type    { return PingFailedType }
func (e PingFailed) String() string     { return fmt.Sprintf("ping failed: %s", e.Reason) }
func (e PingFailed) Fields() log.Fields { return map[string]interface{}{"error": e.Reason} }
