package stream

import (
	"context"
	"errors"
	"fmt"

	"github.com/modernice/goes/event"
)

var (
	// ErrClosed is returned by a Stream when trying to read from it after it
	// has been closed.
	ErrClosed = errors.New("stream closed")
)

type stream struct {
	events []event.Event
	event  event.Event
	pos    int
	err    error
	closed chan struct{}
}

// New returns an in-memory Stream filled with the provided events.
func New(events ...event.Event) event.Stream {
	return &stream{
		events: events,
		closed: make(chan struct{}),
	}
}

// All iterates over the Stream s and returns its Events. If a call to
// s.Next causes an error, the already fetched Events and that error are
// returned. All automatically calls s.Close(ctx) when done.
func All(ctx context.Context, s event.Stream) (events []event.Event, err error) {
	defer func() {
		if cerr := s.Close(ctx); cerr != nil {
			if err != nil {
				err = fmt.Errorf("[0] %w\n[1] %s", err, cerr)
				return
			}
			err = cerr
		}
	}()
	for s.Next(ctx) {
		events = append(events, s.Event())
	}
	if err = s.Err(); err != nil {
		return
	}
	return
}

func (c *stream) Next(ctx context.Context) bool {
	select {
	case <-c.closed:
		c.err = ErrClosed
		return false
	default:
		c.err = nil
	}

	if len(c.events) <= c.pos {
		return false
	}
	c.event = c.events[c.pos]
	c.pos++
	return true
}

func (c *stream) Event() event.Event {
	return c.event
}

func (c *stream) Err() error {
	return c.err
}

func (c *stream) Close(context.Context) error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}