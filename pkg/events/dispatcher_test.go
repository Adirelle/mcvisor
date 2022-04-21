package events_test

import (
	"context"
	"testing"
	"time"

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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	go d.Serve(ctx)

	d.Subscribe(ch)

	<-d.Dispatch("foo")
	<-d.Dispatch(payload)

	select {
	case value, open := <-ch:
		if !open {
			t.Error("channel has been closed")
		} else if value != payload {
			t.Errorf("payload mismatch: %d", value)
		}
	case <-ctx.Done():
		t.Error("timed out")
	}
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	payload := 10

	d := events.NewDispatcher()
	ch := events.MakeHandler[int]()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	go d.Serve(ctx)

	sub := d.Subscribe(ch)
	sub.Cancel()

	<-d.Dispatch("foo")
	<-d.Dispatch(payload)

	select {
	case value := <-ch:
		t.Errorf("unexpected value: %d", value)
	case <-ctx.Done():
	}
}
