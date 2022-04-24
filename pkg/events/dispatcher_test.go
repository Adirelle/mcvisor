package events_test

import (
	"testing"

	"github.com/Adirelle/mcvisor/pkg/events"
	"github.com/apex/log"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestBasic(t *testing.T) {
	t.Parallel()
	payload := 10

	d := events.NewDispatcher()
	ch := events.MakeHandler[int]()

	d.Subscribe(ch)

	d.Dispatch("foo")
	d.Dispatch(payload)

	value, open := <-ch
	if !open {
		t.Error("channel has been closed")
	} else if value != payload {
		t.Errorf("payload mismatch: %d", value)
	}
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	payload := 10

	d := events.NewDispatcher()
	ch := events.MakeHandler[int]()

	sub := d.Subscribe(ch)
	sub.Cancel()

	d.Dispatch("foo")
	d.Dispatch(payload)

	select {
	case value := <-ch:
		t.Errorf("unexpected value: %d", value)
	default:
	}
}
