package minecraft

import (
	"context"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/discord"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/apex/log"
	"github.com/thejerf/suture/v4"
)

type (
	Controller struct {
		dispatcher *events.Dispatcher
		ctl        Control
		target     Target
		status     Status

		commands chan *commands.Command
		statuses chan Status
	}

	Target string

	Control interface {
		Start()
		Stop()
	}
)

const (
	StartCommand    commands.Name = "start"
	StopCommand     commands.Name = "stop"
	RestartCommand  commands.Name = "restart"
	ShutdownCommand commands.Name = "shutdown"

	StartTarget    Target = "start"
	StopTarget     Target = "stop"
	RestartTarget  Target = "restart"
	ShutdownTarget Target = "shutdown"
)

var (
	commandTargets = map[commands.Name]Target{
		StartCommand:    StartTarget,
		StopCommand:     StopTarget,
		RestartCommand:  RestartTarget,
		ShutdownCommand: ShutdownTarget,
	}

	SystemShutdown = &commands.Command{Name: ShutdownCommand}
)

func init() {
	commands.Register(StartCommand, "start the server", discord.ControlCategory)
	commands.Register(StopCommand, "stop the server", discord.ControlCategory)
	commands.Register(RestartCommand, "restart the server", discord.ControlCategory)
	commands.Register(ShutdownCommand, "stop the server *and* mcvisor", discord.AdminCategory)
}

func NewController(control Control, dispatcher *events.Dispatcher) *Controller {
	return &Controller{
		dispatcher: dispatcher,
		ctl:        control,
		target:     StartTarget,
		status:     Stopped,
		commands:   events.MakeHandler[*commands.Command](),
		statuses:   events.MakeHandler[Status](),
	}
}

func (c *Controller) Serve(ctx context.Context) error {
	defer c.dispatcher.Subscribe(c.commands).Cancel()
	defer c.dispatcher.Subscribe(c.statuses).Cancel()

	done := ctx.Done()

	for {
		if err := c.tick(); err != nil {
			return err
		}

		select {
		case cmd := <-c.commands:
			if newTarget, found := commandTargets[cmd.Name]; found {
				c.setTarget(newTarget)
			}
		case newStatus := <-c.statuses:
			c.setStatus(newStatus)
		case <-done:
			done = nil
			c.setTarget(ShutdownTarget)
		}
	}
}

func (c *Controller) tick() error {
	switch c.target {
	case RestartTarget:
		if c.status == Stopped {
			c.ctl.Start()
			c.setTarget(StartTarget)
		} else if c.status != Stopping {
			c.ctl.Stop()
		}
	case StartTarget:
		if c.status == Stopped {
			c.ctl.Start()
		}
	case ShutdownTarget:
		if c.status == Stopped {
			return suture.ErrTerminateSupervisorTree
		}
		fallthrough
	case StopTarget:
		if c.status != Stopped && c.status != Stopping {
			c.ctl.Stop()
		}
	}
	return nil
}

func (c *Controller) setTarget(newTarget Target) {
	if newTarget == c.target {
		return
	}
	c.target = newTarget
	log.WithField("target", newTarget).Debug("controller.target")
	c.dispatcher.Dispatch(newTarget)
	return
}

func (c *Controller) setStatus(newStatus Status) {
	if newStatus == c.status {
		return
	}
	c.status = newStatus
	log.WithField("status", newStatus).Debug("controller.status")
}
