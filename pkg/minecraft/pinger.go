package minecraft

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/apex/log"
	properties "github.com/dmotylev/goproperties"
	"github.com/millkhan/mcstatusgo/v2"
)

type (
	Pinger struct {
		*Config
		*events.Dispatcher
		lastPing PingerEvent
		commands chan *commands.Command
	}

	pingStrategy interface {
		Ping(when time.Time) PingerEvent
	}

	statusPingStrategy struct {
		*Config
	}

	queryPingSrategy struct {
		*Config
		QueryPort uint16
	}

	nullPingStrategy time.Time

	PingerEvent interface {
		IsSuccess() bool
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

const (
	OnlineCommand commands.Name = "online"
)

var (
	// Interface checks
	_ PingerEvent  = (*PingSucceeded)(nil)
	_ PingerEvent  = (*PingFailed)(nil)
	_ PingerEvent  = PingDisabled(time.Now())
	_ pingStrategy = (*statusPingStrategy)(nil)
	_ pingStrategy = (*queryPingSrategy)(nil)
	_ pingStrategy = nullPingStrategy(time.Now())
)

func init() {
	commands.Register(OnlineCommand, "list online players", discord.QueryCategory)
}

func NewPinger(config *Config, dispatcher *events.Dispatcher) *Pinger {
	return &Pinger{
		Config:     config,
		Dispatcher: dispatcher,
		commands:   events.MakeHandler[*commands.Command](),
	}
}

func (p *Pinger) Serve(ctx context.Context) error {
	pingStrategy, err := p.getPingStrategy()
	if err != nil {
		log.WithError(err).WithField("path", p.AbsServerProperties()).Error("pinger.config")
		return err
	}

	defer p.Subscribe(p.commands).Cancel()

	ticker := time.NewTicker(p.PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case cmd := <-p.commands:
			if cmd.Name == OnlineCommand {
				p.lastPing.writeReport(cmd.Reply)
			}
		case when := <-ticker.C:
			p.lastPing = pingStrategy.Ping(when)
			log.WithField("result", p.lastPing).Debug("pinger.update")
			p.Dispatch(p.lastPing)
		}
	}
}

func (p *Pinger) getPingStrategy() (pingStrategy, error) {
	props, err := properties.Load(p.AbsServerProperties())
	if err != nil {
		return nil, err
	}

	p.ServerPort = uint16(props.Int("server-port", int64(p.ServerPort)))

	if props.Bool("enable-query", false) {
		port := uint16(props.Int("query.port", int64(p.ServerPort)))
		return &queryPingSrategy{p.Config, port}, nil
	}

	if props.Bool("enable-status", false) {
		return &statusPingStrategy{p.Config}, nil
	}

	return nullPingStrategy(time.Now()), nil
}

func (p *queryPingSrategy) Ping(when time.Time) PingerEvent {
	log.Debug("pinger.ping.fullQuery")
	if response, err := mcstatusgo.FullQuery(p.ServerHost, p.QueryPort, p.ConnectionTimeout, p.ResponseTimeout); err == nil {
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

func (p *statusPingStrategy) Ping(when time.Time) PingerEvent {
	log.Debug("pinger.ping.status")
	if response, err := mcstatusgo.Status(p.ServerHost, p.ServerPort, p.ConnectionTimeout, p.ResponseTimeout); err == nil {
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

func (p nullPingStrategy) Ping(_ time.Time) PingerEvent {
	return PingDisabled(p)
}

func (PingSucceeded) IsSuccess() bool {
	return true
}

func (p *PingSucceeded) Fields() log.Fields {
	return log.Fields{
		"latency":        p.Latency,
		"players.online": p.OnlinePlayers,
		"players.max":    p.MaxPlayers,
		"players.list":   p.PlayerList,
	}
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

func (PingFailed) IsSuccess() bool {
	return false
}

func (p *PingFailed) Fields() log.Fields {
	return log.Fields{"error": p.Reason}
}

func (p *PingFailed) writeReport(writer io.Writer) (err error) {
	_, err = io.WriteString(writer, "**last ping failed**")
	return
}

func (PingDisabled) IsSuccess() bool {
	return false
}

func (PingDisabled) Fields() log.Fields {
	return nil
}

func (PingDisabled) writeReport(writer io.Writer) (err error) {
	_, err = io.WriteString(writer, "**both status and query are disabled in server configuration**")
	return
}
