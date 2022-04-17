package minecraft

import (
	"context"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/apex/log"
	"github.com/thejerf/suture/v4"
)

type (
	Controller struct {
		events.HandlerBase
		events.Dispatcher
		ctl Control
	}

	Target string

	Control interface {
		Start()
		Stop()
	}

	TargetChanged struct {
		Target
	}

	systemShutdown int
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

	SystemShutdown events.Event = systemShutdown(0)

	_10 events.Event = (*TargetChanged)(nil)
)

func init() {
	commands.Register(StartCommand, "start the server", permissions.ControlCategory)
	commands.Register(StopCommand, "stop the server", permissions.ControlCategory)
	commands.Register(RestartCommand, "restart the server", permissions.ControlCategory)
	commands.Register(ShutdownCommand, "stop the server *and* mcvisor", permissions.AdminCategory)
}

func NewController(control Control, dispatcher events.Dispatcher) *Controller {
	c := &Controller{
		HandlerBase: events.MakeHandlerBase(),
		Dispatcher:  dispatcher,
		ctl:         control,
	}
	dispatcher.Add(c)
	return c
}

func (c *Controller) Serve(ctx context.Context) error {
	status := Stopped
	target := StopTarget
	newStatus := status
	newTarget := StartTarget
	done := ctx.Done()

	for {
		if newStatus != status || newTarget != target {
			if newStatus != status {
				status = newStatus
				log.WithField("status", status).Debug("controller.status")
			}
			if newTarget != target {
				target = newTarget
				log.WithField("target", target).Debug("controller.target")
				c.Dispatch(&TargetChanged{target})
			}

			switch target {
			case RestartTarget:
				if status == Stopped {
					newTarget = StartTarget
					c.ctl.Start()
				} else if status != Stopping {
					c.ctl.Stop()
				}
			case StartTarget:
				if status == Stopped {
					c.ctl.Start()
				}
			case ShutdownTarget:
				if status == Stopped {
					return suture.ErrTerminateSupervisorTree
				}
				fallthrough
			case StopTarget:
				if status != Stopped && status != Stopping {
					c.ctl.Stop()
				}
			}
		}

		select {
		case event := <-c.HandlerBase:
			switch typedEvent := event.(type) {
			case StatusChanged:
				newStatus = typedEvent.Status()
			case systemShutdown:
				newTarget = ShutdownTarget
			case *commands.Command:
				if target, found := commandTargets[typedEvent.Name]; found {
					newTarget = target
				}
			}
		case <-done:
			done = nil
			newTarget = ShutdownTarget
		}
	}
}

func (t *TargetChanged) Fields() log.Fields {
	return log.Fields{"target": t.Target}
}

func (systemShutdown) Fields() log.Fields {
	return nil
}
