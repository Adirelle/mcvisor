package minecraft

import (
	"context"
	"fmt"
	"io"
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
		serverPropertiesPath string
		events.Dispatcher
		events.HandlerBase
		lastPing PingerEvent
	}

	pinger interface {
		Ping(when time.Time) PingerEvent
	}

	statusPinger uint16
	queryPinger  uint16
	nullPinger   time.Time

	PingerEvent interface {
		events.Event
		writeReport(io.Writer) error
	}

	PingSucceeded struct {
		When          time.Time
		Latency       time.Duration
		MaxPlayers    uint
		OnlinePlayers uint
		PlayerList    []string
	}

	PingFailed struct {
		When   time.Time
		Reason error
	}

	PingDisabled time.Time
)

var (
	// Interface checks
	_1 PingerEvent = (*PingSucceeded)(nil)
	_2 PingerEvent = (*PingFailed)(nil)
	_3 PingerEvent = PingDisabled(time.Now())
	_4 pinger      = statusPinger(0)
	_5 pinger      = queryPinger(0)
	_6 pinger      = nullPinger(time.Now())
)

const (
	OnlineCommand commands.Name = "online"
)

func init() {
	commands.Register(OnlineCommand, "list online players", permissions.QueryCategory)
}

func NewPinger(conf Config, dispatcher events.Dispatcher) *Pinger {
	return &Pinger{
		serverPropertiesPath: conf.AbsServerProperties(),
		Dispatcher:           dispatcher,
		HandlerBase:          events.MakeHandlerBase(),
	}
}

func (p *Pinger) Serve(ctx context.Context) error {
	pinger, err := p.newPinger()
	if err != nil {
		log.WithError(err).WithField("path", p.serverPropertiesPath).Error("pinger.config")
		return err
	}

	ticker := time.NewTicker(PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-p.HandlerBase:
			commands.OnCommand(OnlineCommand, event, p.handleOnlineCommand)
		case when := <-ticker.C:
			p.lastPing = pinger.Ping(when)
			log.WithField("result", p.lastPing).Debug("pinger.update")
			p.DispatchEvent(p.lastPing)
		}
	}
}

func (p *Pinger) newPinger() (pinger, error) {
	props, err := properties.Load(p.serverPropertiesPath)
	if err != nil {
		return nil, err
	}

	serverPort := uint16(props.Int("server-port", DefaultServerPort))

	if props.Bool("enable-query", false) {
		port := uint16(props.Int("query.port", int64(serverPort)))
		return queryPinger(port), nil
	}

	if props.Bool("enable-status", false) {
		return statusPinger(serverPort), nil
	}

	return nullPinger(time.Now()), nil
}

func (p *Pinger) handleOnlineCommand(command *commands.Command) error {
	return p.lastPing.writeReport(command.Reply)
}

func (p queryPinger) Ping(when time.Time) PingerEvent {
	log.Debug("pinger.ping.fullQuery")
	if response, err := mcstatusgo.FullQuery(ServerHost, uint16(p), ConnectionTimeout, ResponseTimeout); err == nil {
		return &PingSucceeded{
			When:          when,
			Latency:       response.Latency,
			MaxPlayers:    uint(response.Players.Max),
			OnlinePlayers: uint(response.Players.Online),
			PlayerList:    response.Players.PlayerList,
		}
	} else {
		return &PingFailed{when, err}
	}
}

func (p statusPinger) Ping(when time.Time) PingerEvent {
	log.Debug("pinger.ping.status")
	if response, err := mcstatusgo.Status(ServerHost, uint16(p), ConnectionTimeout, ResponseTimeout); err == nil {
		return &PingSucceeded{
			When:          when,
			Latency:       response.Latency,
			MaxPlayers:    uint(response.Players.Max),
			OnlinePlayers: uint(response.Players.Online),
			PlayerList:    nil,
		}
	} else {
		return &PingFailed{when, err}
	}
}

func (p nullPinger) Ping(_ time.Time) PingerEvent {
	return PingDisabled(p)
}

func (p *PingSucceeded) writeReport(writer io.Writer) error {
	_, _ = fmt.Fprintf(writer, "Online players: %d/%d (<t:%d:R>)", p.OnlinePlayers, p.MaxPlayers, p.When.Unix())
	if len(p.PlayerList) > 0 {
		for _, name := range p.PlayerList {
			fmt.Fprintf(writer, "\n\n- %s", name)
		}
	}
	return nil
}

func (p *PingFailed) writeReport(writer io.Writer) (err error) {
	_, err = io.WriteString(writer, "**last ping failed**")
	return
}

func (PingDisabled) writeReport(writer io.Writer) (err error) {
	_, err = io.WriteString(writer, "**both status and query are disabled in server configuration**")
	return
}
