package minecraft

import (
	"context"

	"github.com/Adirelle/mcvisor/pkg/events"
)

type (
	Controller struct {
		ctl Control

		current Status
		process func()
	}

	Control interface {
		Start()
		Stop()
	}
)

func NewController(control Control) (c *Controller) {
	c = &Controller{ctl: control, current: Stopped}
	c.process = c.Start
	return
}

func (c *Controller) Serve(ctx context.Context) error {
	c.process()
	<-ctx.Done()
	return nil
}

func (c *Controller) HandleEvent(event events.Event) {
	if statusChanged, ok := event.(StatusChanged); ok {
		if statusChanged.Status == c.current {
			return
		}
		c.current = statusChanged.Status
		c.process()
	}
}

func (c *Controller) Start() {
	if c.current == Stopped {
		c.ctl.Start()
	}
}
