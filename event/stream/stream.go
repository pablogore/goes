package stream

import (
	"context"

	"github.com/modernice/goes/event"
	"github.com/modernice/goes/internal/xerror"
)

// Slice takes a slice of Events and returns an Event channel which is filled
// with the given Events in a separate goroutine and closed afterwards.
func Slice(events ...event.Event) <-chan event.Event {
	out := make(chan event.Event)
	go func() {
		defer close(out)
		for _, evt := range events {
			out <- evt
		}
	}()
	return out
}

// Drain drains the given Event channel and returns its Events.
//
// Drain accepts optional error channels which will cause Drain to fail on any
// error. When Drain encounters an error from any of the error channels, the
// already drained Events and that error are returned. Similarly, when ctx is
// canceled, the drained Events and ctx.Err() are returned.
//
// It is not necessary for the error channels to be closed by the caller because
// Drain automatically cleans up its goroutines when returning the result.
func Drain(ctx context.Context, events <-chan event.Event, errs ...<-chan error) ([]event.Event, error) {
	errChan, stop := xerror.FanIn(errs...)
	defer stop()

	out := make([]event.Event, 0, len(events))

	for {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				break
			}
			return out, err
		case evt, ok := <-events:
			if !ok {
				return out, nil
			}
			out = append(out, evt)
		}
	}
}
