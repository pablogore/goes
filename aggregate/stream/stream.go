package stream

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/aggregate/internal/softdelete"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/helper/streams"
)

// Option is a stream option.
type Option func(*options)

type options struct {
	isSorted            bool
	isGrouped           bool
	validateConsistency bool
	withSoftDeleted     bool
	filters             []func(event.Event) bool
	streamErrors        []<-chan error
}

type stream struct {
	options

	stream     <-chan event.Evt[any]
	inErrors   <-chan error
	stopErrors func()

	acceptDone chan struct{}

	events   chan event.Event
	complete chan job

	groupReqs chan groupRequest

	out       chan aggregate.History
	outErrors chan error
}

type job struct {
	name string
	id   uuid.UUID
}

type groupRequest struct {
	job
	out chan []event.Event
}

type applier struct {
	job

	apply func(aggregate.Aggregate)
}

// Errors returns an Option that provides a Stream with error channels. A Stream
// will cancel its operation as soon as an error can be received from one of the
// error channels.
func Errors(errs ...<-chan error) Option {
	return func(opts *options) {
		opts.streamErrors = append(opts.streamErrors, errs...)
	}
}

// Sorted returns an Option that optimizes aggregate builds by giving the
// Stream information about the order of incoming events from the streams.New.
//
// When Sorted is disabled (which it is by default), the Stream sorts the
// collected events for a specific aggregate by the AggregateVersion of the
// Events before applying them to the aggregate.
//
// Enable this option only if the underlying streams.New guarantees that
// incoming events are sorted by aggregateVersion.
func Sorted(v bool) Option {
	return func(opts *options) {
		opts.isSorted = v
	}
}

// Grouped returns an Option that optimizes aggregate builds by giving the
// Stream information about the order of incoming events from the streams.New.
//
// When Grouped is disabled, the Stream has to wait for the streams.New to be
// drained before it can be sure no more events will arrive for a specific
// Aggregate. When Grouped is enabled, the Stream knows when all events for an
// Aggregate have been received and can therefore return the aggregate as soon
// as its last event has been received and applied.
//
// Grouped is disabled by default and should only be enabled if the correct
// order of events is guaranteed by the underlying streams.New. events are
// correctly ordered only if they're sequentially grouped by aggregate. Sorting
// within a group of events does not matter if IsSorted is disabled (which it is
// by default). When IsSorted is enabled, events within a group must be ordered
// by aggregateVersion.
//
// An example for correctly ordered events (with IsSorted disabled):
//
// 	name="foo" id="BBXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=2
// 	name="foo" id="BBXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=1
// 	name="foo" id="BBXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=4
// 	name="foo" id="BBXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=3
// 	name="bar" id="AXXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=1
// 	name="bar" id="AXXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=2
// 	name="bar" id="AXXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=3
// 	name="bar" id="AXXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=4
// 	name="foo" id="AAXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=4
// 	name="foo" id="AAXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=3
// 	name="foo" id="AAXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=2
// 	name="foo" id="AAXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=1
// 	name="bar" id="BXXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=2
// 	name="bar" id="BXXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=1
// 	name="bar" id="BXXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=3
// 	name="bar" id="BXXXXXXX-XXXX-XXXX-XXXXXXXXXXXX" version=4
func Grouped(v bool) Option {
	return func(opts *options) {
		opts.isGrouped = v
	}
}

// ValidateConsistency returns an Option that optimizes aggregate builds by
// controlling if the consistency of events is validated before building an
// Aggregate from those events.
//
// This option is enabled by default and should only be disabled if the
// consistency of events is guaranteed by the underlying streams.New or if it's
// explicitly desired to put an aggregate into an invalid state.
func ValidateConsistency(v bool) Option {
	return func(opts *options) {
		opts.validateConsistency = v
	}
}

// Filter returns an Option that filters incoming events before they're handled
// by the Stream. events are passed to every fn in fns until a fn returns false.
// If any of fns returns false, the event is discarded by the Stream.
func Filter(fns ...func(event.Event) bool) Option {
	return func(opts *options) {
		opts.filters = append(opts.filters, fns...)
	}
}

// WithSoftDeletes returns an Option that specifies if the stream should return
// soft-deleted aggregates in the returned History stream. Soft-deleted aggregates
// are by default excluded from the result.
func WithSoftDeleted(v bool) Option {
	return func(opts *options) {
		opts.withSoftDeleted = v
	}
}

// New takes a channel of events and returns both a channel of aggregate
// Histories and an error channel. A History apply itself on an aggregate to
// build the current state of the aggregate.
//
// Use the Drain function to get the Histories as a slice and a single error:
//
//	var events <-chan event.Event
//	str, errs := stream.New(events)
//	histories, err := streams.Drain(context.TODO(), str, errs)
//	// handle err
//	for _, h := range histories {
//		foo := newFoo(h.AggregateID())
//		h.Apply(foo)
//	}
func New(ctx context.Context, events <-chan event.Event, opts ...Option) (<-chan aggregate.History, <-chan error) {
	return NewOf[any](ctx, events, opts...)
}

// NewOf takes a channel of events and returns both a channel of aggregate
// Histories and an error channel. A History apply itself on an aggregate to
// build the current state of the aggregate.
//
// Use the Drain function to get the Histories as a slice and a single error:
//
//	var events <-chan event.Event
//	str, errs := stream.New(events)
//	histories, err := streams.Drain(context.TODO(), str, errs)
//	// handle err
//	for _, h := range histories {
//		foo := newFoo(h.AggregateID())
//		h.Apply(foo)
//	}
func NewOf[D any, Event event.Of[D]](ctx context.Context, events <-chan Event, opts ...Option) (<-chan aggregate.History, <-chan error) {
	if events == nil {
		evts := make(chan Event)
		close(evts)
		events = evts
	}

	aes := stream{
		options:    options{validateConsistency: true},
		stream:     streams.Map(ctx, events, func(e Event) event.Evt[any] { return event.Any[D](e) }),
		acceptDone: make(chan struct{}),
		events:     make(chan event.Event),
		complete:   make(chan job),
		groupReqs:  make(chan groupRequest),
		out:        make(chan aggregate.History),
		outErrors:  make(chan error),
	}
	for _, opt := range opts {
		opt(&aes.options)
	}

	aes.inErrors, aes.stopErrors = streams.FanIn(aes.streamErrors...)

	go aes.acceptEvents()
	go aes.groupEvents()
	go aes.sortEvents()

	return aes.out, aes.outErrors
}

func (s *stream) acceptEvents() {
	defer close(s.complete)
	defer close(s.acceptDone)
	defer close(s.events)
	defer s.stopErrors()

	pending := make(map[job]bool)

	var prev job
L:
	for {
		select {
		case err, ok := <-s.inErrors:
			if !ok {
				s.inErrors = nil
				break
			}
			s.outErrors <- fmt.Errorf("event stream: %w", err)
			break L
		case evt, ok := <-s.stream:
			if !ok {
				break L
			}

			if s.shouldDiscard(evt) {
				break
			}

			s.events <- evt

			id, name, _ := evt.Aggregate()

			j := job{
				name: name,
				id:   id,
			}

			pending[j] = true

			if s.isGrouped && prev.name != "" && prev != j {
				s.complete <- prev
				delete(pending, prev)
			}

			prev = j
		}
	}

	for j := range pending {
		s.complete <- j
	}
}

func (s *stream) shouldDiscard(evt event.Event) bool {
	for _, fn := range s.filters {
		if !fn(evt) {
			return true
		}
	}
	return false
}

func (s *stream) groupEvents() {
	groups := make(map[job][]event.Event)
	events := s.events
	groupReqs := s.groupReqs
	for {
		if events == nil && groupReqs == nil {
			return
		}

		select {
		case evt, ok := <-events:
			if !ok {
				events = nil
				break
			}
			id, name, _ := evt.Aggregate()
			j := job{name, id}
			groups[j] = append(groups[j], evt)
		case req, ok := <-groupReqs:
			if !ok {
				groupReqs = nil
				break
			}
			req.out <- groups[req.job]
			delete(groups, req.job)
		}
	}
}

func (s *stream) sortEvents() {
	defer close(s.out)
	defer close(s.outErrors)
	defer close(s.groupReqs)

	for j := range s.complete {
		req := groupRequest{
			job: j,
			out: make(chan []event.Event),
		}
		s.groupReqs <- req
		events := <-req.out

		if !s.isSorted {
			events = event.Sort(events, event.SortAggregateVersion, event.SortAsc)
		}

		if s.validateConsistency {
			a := aggregate.New(j.name, j.id)
			if err := aggregate.ValidateConsistency(a, events); err != nil {
				s.outErrors <- err
				continue
			}
		}

		if !s.withSoftDeleted && softdelete.SoftDeleted(events) {
			continue
		}

		s.out <- applier{
			job:   j,
			apply: func(a aggregate.Aggregate) { aggregate.ApplyHistory(a, events) },
		}
	}
}

func (a applier) Aggregate() aggregate.Ref {
	return aggregate.Ref{Name: a.name, ID: a.id}
}

func (a applier) Apply(ag aggregate.Aggregate) {
	a.apply(ag)
}
