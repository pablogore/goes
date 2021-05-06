package project

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"sync"
	stdtime "time"

	"github.com/google/uuid"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/event/query"
	"github.com/modernice/goes/event/query/time"
	"github.com/modernice/goes/internal/unique"
)

// A Job is provided by a Projector when a projection should be run.
type Job interface {
	// Context returns the projection Context.
	Context() context.Context

	// Events returns the Events from the Job.
	Events(context.Context) ([]event.Event, error)

	// EventsOf returns the Events from the Job that have one of the given names.
	EventsOf(context.Context, ...string) ([]event.Event, error)

	// EventsFor returns the Events that would be applied to the given
	// projection. If the projection provides a `LatestEventTime` method, it is
	// used to only query Events that happened after that time.
	EventsFor(context.Context, EventApplier, ...ApplyOption) ([]event.Event, error)

	// Aggregates returns a map of Aggregate names to UUIDs, extracted from the
	// Events of the Job.
	Aggregates(context.Context) (map[string][]uuid.UUID, error)

	// AggregatesOf returns the UUIDs of Aggregates that have one of the given
	// names, extracted from the Events of the Job.
	AggregatesOf(context.Context, ...string) ([]uuid.UUID, error)

	// Aggregate returns the first UUID of an Aggregate with the given name,
	// extracted from the Events of the Job.
	//
	// If no Event belongs to an Aggregate witht that name, uuid.Nil is returned.
	Aggregate(context.Context, string) (uuid.UUID, error)

	// Apply applies the projection on an EventApplier, which is usually a type
	// that embeds *Projection.
	Apply(context.Context, EventApplier, ...ApplyOption) error
}

// ApplyOption is an option for applying a projection Job.
type ApplyOption func(*applyConfig)

type applyConfig struct {
	fromBase bool
}

type continuousJob struct {
	*baseJob

	events []event.Event
}

type periodicJob struct {
	*baseJob
}

type baseJob struct {
	*cache

	ctx        context.Context
	cfg        subscribeConfig
	store      event.Store
	eventNames []string
	query      event.Query
}

type cache struct {
	sync.Mutex

	cache map[[32]byte][]event.Event
}

type cacheHasher struct {
	cfg   applyConfig
	query event.Query
}

func newContinuousJob(
	ctx context.Context,
	cfg subscribeConfig,
	store event.Store,
	query event.Query,
	events []event.Event,
	eventNames []string,
) *continuousJob {
	return &continuousJob{
		baseJob: &baseJob{
			cache:      newCache(),
			ctx:        ctx,
			cfg:        cfg,
			store:      store,
			query:      query,
			eventNames: eventNames,
		},
		events: events,
	}
}

func (j *continuousJob) Events(ctx context.Context) ([]event.Event, error) {
	return j.EventsFor(ctx, nil)
}

func (j *continuousJob) EventsOf(ctx context.Context, names ...string) ([]event.Event, error) {
	events, err := j.Events(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]event.Event, 0, len(events))
	for _, evt := range events {
		for _, name := range names {
			if evt.Name() == name {
				filtered = append(filtered, evt)
				break
			}
		}
	}

	return filtered, nil
}

func (j *continuousJob) EventsFor(ctx context.Context, p EventApplier, opts ...ApplyOption) ([]event.Event, error) {
	cfg := configureApply(opts...)

	if !cfg.fromBase && j.events != nil {
		return j.events, nil
	}

	return j.baseJob.EventsFor(ctx, p, opts...)
}

func (j *continuousJob) Aggregates(ctx context.Context) (map[string][]uuid.UUID, error) {
	if ctx == nil {
		ctx = j.ctx
	}

	events, err := j.Events(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[string][]uuid.UUID, len(events))
	for _, evt := range events {
		if evt.AggregateName() != "" {
			out[evt.AggregateName()] = append(out[evt.AggregateName()], evt.AggregateID())
		}
	}

	for name, ids := range out {
		out[name] = unique.UUID(ids...)
	}

	return out, nil
}

func (j *continuousJob) AggregatesOf(ctx context.Context, names ...string) ([]uuid.UUID, error) {
	if ctx == nil {
		ctx = j.ctx
	}

	aggregates, err := j.Aggregates(ctx)
	if err != nil {
		return nil, err
	}

	var out []uuid.UUID
	for _, name := range names {
		out = append(out, aggregates[name]...)
	}

	return out, nil
}

func (j *continuousJob) Aggregate(ctx context.Context, name string) (uuid.UUID, error) {
	if ctx == nil {
		ctx = j.ctx
	}

	ids, err := j.AggregatesOf(ctx, name)
	if err != nil {
		return uuid.Nil, err
	}

	if len(ids) == 0 {
		return uuid.Nil, nil
	}

	return ids[0], nil
}

func (j *continuousJob) Apply(ctx context.Context, p EventApplier, opts ...ApplyOption) error {
	if ctx == nil {
		ctx = j.ctx
	}

	events, err := j.EventsFor(ctx, p, opts...)
	if err != nil {
		return err
	}

	return Apply(events, p)
}

func newPeriodicJob(ctx context.Context, cfg subscribeConfig, store event.Store, eventNames []string, query event.Query) *periodicJob {
	return &periodicJob{
		baseJob: &baseJob{
			cache:      newCache(),
			ctx:        ctx,
			cfg:        cfg,
			store:      store,
			eventNames: eventNames,
			query:      query,
		},
	}
}

func (j *baseJob) Context() context.Context {
	return j.ctx
}

func (j *baseJob) Events(ctx context.Context) ([]event.Event, error) {
	return j.EventsFor(ctx, nil)
}

func (j *periodicJob) EventsOf(ctx context.Context, names ...string) ([]event.Event, error) {
	events, err := j.Events(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]event.Event, 0, len(events))
	for _, evt := range events {
		for _, name := range names {
			if evt.Name() == name {
				filtered = append(filtered, evt)
				break
			}
		}
	}

	return filtered, nil
}

func (j *periodicJob) Aggregates(ctx context.Context) (map[string][]uuid.UUID, error) {
	if ctx == nil {
		ctx = j.ctx
	}

	events, err := j.Events(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[string][]uuid.UUID, len(events))
	for _, evt := range events {
		if evt.AggregateName() != "" {
			out[evt.AggregateName()] = append(out[evt.AggregateName()], evt.AggregateID())
		}
	}

	for name, ids := range out {
		out[name] = unique.UUID(ids...)
	}

	return out, nil
}

func (j *periodicJob) AggregatesOf(ctx context.Context, names ...string) ([]uuid.UUID, error) {
	if ctx == nil {
		ctx = j.ctx
	}

	aggregates, err := j.Aggregates(ctx)
	if err != nil {
		return nil, err
	}

	var out []uuid.UUID
	for _, name := range names {
		out = append(out, aggregates[name]...)
	}

	return out, nil
}

func (j *periodicJob) Aggregate(ctx context.Context, name string) (uuid.UUID, error) {
	if ctx == nil {
		ctx = j.ctx
	}

	ids, err := j.AggregatesOf(ctx, name)
	if err != nil {
		return uuid.Nil, err
	}

	if len(ids) == 0 {
		return uuid.Nil, nil
	}

	return ids[0], nil
}

func (j *baseJob) Apply(ctx context.Context, p EventApplier, opts ...ApplyOption) error {
	if ctx == nil {
		ctx = j.ctx
	}

	events, err := j.EventsFor(ctx, p, opts...)
	if err != nil {
		return err
	}

	return Apply(events, p)
}

func (j *baseJob) EventsFor(ctx context.Context, p EventApplier, opts ...ApplyOption) ([]event.Event, error) {
	start := stdtime.Now()
	defer func() {
		log.Printf("Fetching Events took %v\n", stdtime.Since(start))
	}()

	cfg := configureApply(opts...)

	if ctx == nil {
		ctx = j.ctx
	}

	return j.eventsFor(ctx, p, cfg, j.buildQuery(p, cfg))
}

func (j *baseJob) buildQuery(p EventApplier, cfg applyConfig) event.Query {
	queryOpts := []query.Option{
		query.Name(j.eventNames...),
		query.SortByMulti(
			event.SortOptions{Sort: event.SortTime, Dir: event.SortAsc},
			event.SortOptions{Sort: event.SortAggregateName, Dir: event.SortAsc},
			event.SortOptions{Sort: event.SortAggregateID, Dir: event.SortAsc},
			event.SortOptions{Sort: event.SortAggregateVersion, Dir: event.SortAsc},
		),
	}

	if p, ok := p.(latestEventTimeProvider); ok && !cfg.fromBase {
		queryOpts = append(queryOpts, query.Time(time.After(p.LatestEventTime())))
	}

	return query.Merge(query.New(queryOpts...), j.cfg.filter, j.query)
}

func (j *baseJob) eventsFor(ctx context.Context, proj EventApplier, cfg applyConfig, q event.Query) ([]event.Event, error) {
	if ctx == nil {
		ctx = j.ctx
	}

	return j.cache.ensure(ctx, cfg, q, proj, func(ctx context.Context) ([]event.Event, error) {
		str, errs, err := j.store.Query(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("query Events: %w", err)
		}

		events, err := event.Drain(ctx, str, errs)
		if err != nil {
			return nil, fmt.Errorf("drain Events: %w", err)
		}

		return events, nil
	})
}

func newCache() *cache {
	return &cache{cache: make(map[[32]byte][]event.Event)}
}

func (c *cache) ensure(
	ctx context.Context,
	cfg applyConfig,
	query event.Query,
	proj EventApplier,
	fetch func(ctx context.Context) ([]event.Event, error),
) ([]event.Event, error) {
	c.Lock()
	ce := cacheHasher{
		cfg:   cfg,
		query: query,
	}
	h := ce.hash()

	if events, ok := c.cache[h]; ok {
		out := make([]event.Event, len(events))
		copy(out, events)
		c.Unlock()
		return out, nil
	}
	c.Unlock()

	events, err := fetch(ctx)
	if err != nil {
		return events, err
	}

	out := make([]event.Event, len(events))
	copy(out, events)

	c.Lock()
	c.cache[h] = events
	c.Unlock()

	return out, nil
}

func (ce cacheHasher) hash() [32]byte {
	return sha256.Sum256([]byte(fmt.Sprintf("%v", ce)))
}

func configureApply(opts ...ApplyOption) applyConfig {
	var cfg applyConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
