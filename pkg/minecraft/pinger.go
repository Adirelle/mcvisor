package minecraft

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
		*ServerConfig
		*events.Dispatcher
		lastPing PingerEvent
		statuser Statuser
		pings    chan PingerEvent
		strategy pingStrategy
	}

	Statuser interface {
		Status() Status
	}

	pingStrategy interface {
		Ping(when time.Time) PingerEvent
	}

	statusPingStrategy struct {
		*NetworkConfig
	}

	queryPingSrategy struct {
		*NetworkConfig
		QueryPort uint16
	}

	nullPingStrategy time.Time

	PingerEvent interface {
		IsSuccess() bool
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
)

const (
	OnlineCommand commands.Name = "online"
)

var (
	// Interface checks
	_ PingerEvent = (*PingSucceeded)(nil)
	_ PingerEvent = (*PingFailed)(nil)

	_ pingStrategy = (*statusPingStrategy)(nil)
	_ pingStrategy = (*queryPingSrategy)(nil)
	_ pingStrategy = nullPingStrategy(time.Now())

	_ fmt.Stringer = (*PingSucceeded)(nil)
	_ error        = (*PingFailed)(nil)

	builderPool = &sync.Pool{
		New: func() any { return &strings.Builder{} },
	}

	ErrPingDisabled = errors.New("both query and status are disabled server-side")
	ErrPingNever    = errors.New("status unknown")
)

func NewPinger(config *ServerConfig, statuser Statuser, dispatcher *events.Dispatcher) *Pinger {
	p := &Pinger{
		ServerConfig: config,
		Dispatcher:   dispatcher,
		statuser:     statuser,
		pings:        make(chan PingerEvent),
	}
	commands.Register(OnlineCommand, "list online players", discord.QueryCategory, commands.HandlerFunc(p.handleOnlineCommand))
	return p
}

func (p *Pinger) Serve(ctx context.Context) (err error) {
	p.strategy, err = p.getPingStrategy()
	if err != nil {
		log.WithError(err).WithField("path", p.AbsServerProperties()).Error("pinger.config")
		return err
	}

	ticker := time.NewTicker(p.Network.PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case p.lastPing = <-p.pings:
			log.WithField("result", p.lastPing).Debug("pinger.update")
			p.Dispatch(p.lastPing)
		case when := <-ticker.C:
			if p.statuser.Status().IsRunning() {
				go p.Ping(when, ctx)
			} else {
				p.lastPing = &PingFailed{when, ErrPingNever}
			}
		}
	}
}

func (p *Pinger) handleOnlineCommand(cmd *commands.Command) (string, error) {
	switch ping := p.lastPing.(type) {
	case error:
		return "", ping
	case fmt.Stringer:
		return ping.String(), nil
	default:
		return "", nil
	}
}

func (p *Pinger) Ping(when time.Time, ctx context.Context) {
	pingCtx, cleanup := context.WithTimeout(ctx, p.Network.ConnectionTimeout+p.Network.ResponseTimeout)
	defer cleanup()
	ping := p.strategy.Ping(when)
	select {
	case p.pings <- ping:
	case <-pingCtx.Done():
	}
}

func (p *Pinger) getPingStrategy() (pingStrategy, error) {
	props, err := properties.Load(p.AbsServerProperties())
	if err != nil {
		return nil, err
	}

	if p.Network.Host == "" {
		p.Network.Host = props.String("server-ip", "localhost")
	}
	if p.Network.Port == 0 {
		p.Network.Port = uint16(props.Int("server-port", 25565))
	}

	if props.Bool("enable-query", false) {
		port := uint16(props.Int("query.port", int64(p.Network.Port)))
		return &queryPingSrategy{p.Network, port}, nil
	}

	if props.Bool("enable-status", false) {
		return &statusPingStrategy{p.Network}, nil
	}

	return nullPingStrategy(time.Now()), nil
}

func (p *queryPingSrategy) Ping(when time.Time) PingerEvent {
	log.Debug("pinger.ping.fullQuery")
	if response, err := mcstatusgo.FullQuery(p.Host, p.QueryPort, p.ConnectionTimeout, p.ResponseTimeout); err == nil {
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
	if response, err := mcstatusgo.Status(p.Host, p.Port, p.ConnectionTimeout, p.ResponseTimeout); err == nil {
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

func (p nullPingStrategy) Ping(when time.Time) PingerEvent {
	return PingFailed{when, ErrPingDisabled}
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

func (p *PingSucceeded) String() string {
	builder := builderPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		builderPool.Put(builder)
	}()
	_, _ = fmt.Fprintf(builder, "Online players: %d/%d (<t:%d:R>)", p.OnlinePlayers, p.MaxPlayers, p.When.Unix())
	if len(p.PlayerList) > 0 {
		for _, name := range p.PlayerList {
			_, _ = fmt.Fprintf(builder, "\n- %s", name)
		}
	}
	return builder.String()
}

func (PingFailed) IsSuccess() bool {
	return false
}

func (p *PingFailed) Fields() log.Fields {
	return log.Fields{"error": p.Reason}
}

func (p *PingFailed) Error() string {
	return fmt.Sprintf("error: %s", p.Reason.Error())
}
