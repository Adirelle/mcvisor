package minecraft

import (
	"context"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
	"github.com/apex/log"
)

type (
	Controller struct {
		events.HandlerBase
		events.Dispatcher
		ctl    Control
		status Status
		target Target
	}

	Target interface {
		apply(status Status, ctl Control) Target
	}

	Control interface {
		Start()
		Stop()
		Terminate()
	}

	TargetChanged struct {
		Target
	}

	startTarget    string
	stopTarget     string
	restartTarget  string
	shutdownTarget string
)

const (
	StartCommand    commands.Name = "start"
	StopCommand     commands.Name = "stop"
	RestartCommand  commands.Name = "restart"
	ShutdownCommand commands.Name = "shutdown"

	StartTarget    startTarget    = "start"
	StopTarget     stopTarget     = "stop"
	RestartTarget  restartTarget  = "restart"
	ShutdownTarget shutdownTarget = "shutdown"
)

func init() {
	commands.Register(StartCommand, "start the server", permissions.ControlCategory)
	commands.Register(StopCommand, "stop the server", permissions.ControlCategory)
	commands.Register(RestartCommand, "restart the server", permissions.ControlCategory)
	commands.Register(ShutdownCommand, "stop the server *and* mcvisor", permissions.AdminCategory)
}

func NewController(control Control, dispatcher events.Dispatcher) (c *Controller) {
	return &Controller{
		HandlerBase: events.MakeHandlerBase(),
		Dispatcher:  dispatcher,
		ctl:         control,
		status:      Stopped,
		target:      StartTarget,
	}
}

func (c *Controller) Serve(ctx context.Context) error {
	c.iterate()
	return events.Serve(c.HandlerBase, c.HandleEvent, ctx)
}

func (c *Controller) setStatus(status Status) {
	if c.status == status {
		return
	}
	c.status = status
	log.WithField("status", status).Info("controller.status")
	c.iterate()
}

func (c *Controller) SetTarget(target Target) {
	if c.target == target {
		return
	}
	c.target = target
	log.WithField("target", target).Info("controller.target")
	c.DispatchEvent(&TargetChanged{target})
	c.iterate()
}

func (c *Controller) HandleEvent(event events.Event) {
	switch typed := event.(type) {
	case StatusChanged:
		c.setStatus(Status(typed))
	case *commands.Command:
		switch typed.Name {
		case StartCommand:
			c.SetTarget(StartTarget)
		case StopCommand:
			c.SetTarget(StopTarget)
		case RestartCommand:
			c.SetTarget(RestartTarget)
		case ShutdownCommand:
			c.SetTarget(ShutdownTarget)
		}
	}
}

func (c *Controller) iterate() {
	newTarget := c.target.apply(c.status, c.ctl)
	c.SetTarget(newTarget)
}

func (t *TargetChanged) Fields() log.Fields {
	return log.Fields{"target": t.Target}
}

func (t startTarget) apply(status Status, ctl Control) Target {
	if status == Stopped {
		ctl.Start()
	}
	return t
}

func (t stopTarget) apply(status Status, ctl Control) Target {
	switch status {
	case Stopping, Stopped:
		// NOOP
	default:
		ctl.Stop()
	}
	return t
}

func (t restartTarget) apply(status Status, ctl Control) Target {
	switch status {
	case Stopping:
		// NOOP
	case Stopped:
		return StartTarget
	default:
		ctl.Stop()
	}
	return t
}

func (t shutdownTarget) apply(status Status, ctl Control) Target {
	switch status {
	case Stopping:
		// NOOP
	case Stopped:
		ctl.Terminate()
	default:
		ctl.Stop()
	}
	return t
}
