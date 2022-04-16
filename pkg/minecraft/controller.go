package minecraft

import (
	"context"
	"fmt"
	"io"

	"github.com/Adirelle/mcvisor/pkg/commands"
	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/Adirelle/mcvisor/pkg/permissions"
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
		fmt.Stringer
		apply(status Status, ctl Control) Target
	}

	Control interface {
		Start()
		Stop()
		Terminate()
	}

	TargetChanged struct {
		events.Time
		Target
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

	TargetChangedType events.Type = "TargetChanged"

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

func NewController(control Control, dispatcher events.Dispatcher) (c *Controller) {
	return &Controller{HandlerBase: events.MakeHandlerBase(), Dispatcher: dispatcher, ctl: control, status: Stopped, target: StartTarget}
}

func (c *Controller) Serve(ctx context.Context) error {
	c.applyTarget()
	return events.Serve(c.HandlerBase, c.HandleEvent, ctx)
}

func (c *Controller) SetTarget(target Target) {
	if c.target != target {
		c.target = target
		c.DispatchEvent(TargetChanged{events.Now(), target})
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
		_ = command.Reply.Flush()
	}
}

func (c *Controller) applyTarget() {
	c.SetTarget(c.target.apply(c.status, c.ctl))
}

func (t TargetChanged) Type() events.Type { return TargetChangedType }
func (t TargetChanged) String() string    { return t.Target.String() }

func (startTarget) String() string { return "start" }

func (t startTarget) apply(status Status, ctl Control) Target {
	if status == Stopped {
		ctl.Start()
	}
	return t
}

func (stopTarget) String() string { return "stop" }

func (t stopTarget) apply(status Status, ctl Control) Target {
	switch status {
	case Stopping, Stopped:
		// NOOP
	default:
		ctl.Stop()
	}
	return t
}

func (restartTarget) String() string { return "restart" }

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

func (shutdownTarget) String() string { return "shutdown" }

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
