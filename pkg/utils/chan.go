package utils

import (
	"context"
	"errors"
	"time"
)

var ErrChannelClosed = errors.New("channel was closed")

func SendWithTimeout[T any](channel chan<- T, data T, timeout time.Duration) error {
	ctx, cleanup := context.WithTimeout(context.Background(), timeout)
	defer cleanup()
	return SendWithContext(channel, data, ctx)
}

func SendWithContext[T any](channel chan<- T, data T, ctx context.Context) error {
	select {
	case channel <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func RecvWithTimeout[T any](channel <-chan T, timeout time.Duration) (data T, err error) {
	ctx, cleanup := context.WithTimeout(context.Background(), timeout)
	defer cleanup()
	return RecvWithContext(channel, ctx)
}

func RecvWithContext[T any](channel <-chan T, ctx context.Context) (data T, err error) {
	var ok bool
	select {
	case data, ok = <-channel:
		if !ok {
			err = ErrChannelClosed
		}
	case <-ctx.Done():
		err = ctx.Err()
	}
	return
}
