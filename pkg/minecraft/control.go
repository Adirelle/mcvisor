package minecraft

import (
	"context"
	"io"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
)

type (
	Controller struct {
		ctl Control

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

	startTarget    int
	stopTarget     int
	restartTarget  int
	shutdownTarget int
)

const (
	StartCommand    commands.Name = "start"
	StopCommand     commands.Name = "stop"
	RestartCommand  commands.Name = "restart"
	ShutdownCommand commands.Name = "shutdown"

	StartTarget    startTarget    = 0
	StopTarget     stopTarget     = 1
	RestartTarget  restartTarget  = 2
	ShutdownTarget shutdownTarget = 3
)

func init() {
	commands.Register(StartCommand, "start the server", permissions.ControlCategory)
	commands.Register(StopCommand, "stop the server", permissions.ControlCategory)
	commands.Register(RestartCommand, "restart the server", permissions.ControlCategory)
	commands.Register(ShutdownCommand, "stop the server *and* mcvisor", permissions.AdminCategory)
}

func NewController(control Control) (c *Controller) {
	return &Controller{ctl: control, status: Stopped, target: StartTarget}
}

func (c *Controller) Serve(ctx context.Context) error {
	c.applyTarget()
	<-ctx.Done()
	return nil
}

func (c *Controller) SetTarget(target Target) {
	if c.target != target {
		c.target = target
		c.applyTarget()
	}
}

func (c *Controller) HandleEvent(event events.Event) {
	if statusChanged, ok := event.(StatusChanged); ok {
		if statusChanged.Status != c.status {
			c.status = statusChanged.Status
			c.applyTarget()
		}
	}
	if command, ok := event.(commands.Command); ok {
		switch command.Name {
		case StartCommand:
			c.SetTarget(StartTarget)
		case StopCommand:
			c.SetTarget(StopTarget)
		case RestartCommand:
			c.SetTarget(RestartTarget)
		case ShutdownCommand:
			c.SetTarget(ShutdownTarget)
		default:
			return
		}
		_, _ = io.WriteString(command.Reply, "ack")
	}
}

func (c *Controller) applyTarget() {
	c.SetTarget(c.target.apply(c.status, c.ctl))
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
