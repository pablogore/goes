package stream_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/aggregate/stream"
	"github.com/modernice/goes/event"
	estream "github.com/modernice/goes/event/stream"
	"github.com/modernice/goes/event/test"
)

var names = [...]string{
	"foo", "bar", "baz", "foobar",
	"foobaz", "bazbar", "barbaz",
}

type mockAggregate struct {
	aggregate.Aggregate

	a float64
	b string
}

func BenchmarkStream_10A_1E(b *testing.B) {
	benchmark(b, 10, 1)
}

func BenchmarkStream_10A_10E(b *testing.B) {
	benchmark(b, 10, 10)
}

func BenchmarkStream_10A_100E(b *testing.B) {
	benchmark(b, 10, 100)
}

func BenchmarkStream_10A_1000E(b *testing.B) {
	benchmark(b, 10, 1000)
}

func BenchmarkStream_10A_10000E(b *testing.B) {
	benchmark(b, 10, 10000)
}

func BenchmarkStream_10A_100000E(b *testing.B) {
	benchmark(b, 10, 100000)
}

func BenchmarkStream_100A_1E(b *testing.B) {
	benchmark(b, 100, 1)
}

func BenchmarkStream_100A_10E(b *testing.B) {
	benchmark(b, 100, 10)
}

func BenchmarkStream_100A_100E(b *testing.B) {
	benchmark(b, 100, 100)
}

func BenchmarkStream_100A_1000E(b *testing.B) {
	benchmark(b, 100, 1000)
}

func BenchmarkStream_100A_10000E(b *testing.B) {
	benchmark(b, 100, 10000)
}

func BenchmarkStream_1000A_1E(b *testing.B) {
	benchmark(b, 1000, 1)
}

func BenchmarkStream_1000A_10E(b *testing.B) {
	benchmark(b, 1000, 10)
}

func BenchmarkStream_1000A_100E(b *testing.B) {
	benchmark(b, 1000, 100)
}

func BenchmarkStream_1000A_1000E(b *testing.B) {
	benchmark(b, 1000, 1000)
}

func BenchmarkStream_10000A_1E(b *testing.B) {
	benchmark(b, 10000, 1)
}

func BenchmarkStream_10000A_10E(b *testing.B) {
	benchmark(b, 10000, 10)
}

func BenchmarkStream_10000A_100E(b *testing.B) {
	benchmark(b, 10000, 100)
}

func BenchmarkStream_100000A_1E(b *testing.B) {
	benchmark(b, 100000, 1)
}

func BenchmarkStream_100000A_10E(b *testing.B) {
	benchmark(b, 100000, 10)
}

func BenchmarkStream_100000A_100E(b *testing.B) {
	benchmark(b, 100000, 100)
}

func benchmark(b *testing.B, naggregates, nevents int) {
	b.Run("Ungrouped+Unsorted", func(b *testing.B) {
		run(b, naggregates, nevents, false, false)
	})

	b.Run("Grouped+Unsorted", func(b *testing.B) {
		run(b, naggregates, nevents, true, false)
	})

	b.Run("Grouped+Sorted", func(b *testing.B) {
		run(b, naggregates, nevents, true, true)
	})
}

func run(b *testing.B, naggregates, nevents int, grouped, sorted bool) {
	as := makeAggregates(naggregates)
	events := makeEvents(nevents, as, grouped, sorted)
	var opts []stream.Option
	if grouped {
		opts = append(opts, stream.Grouped(true))
	}
	if sorted {
		opts = append(opts, stream.Sorted(true))
	}

	b.ReportAllocs()
	b.ResetTimer()

	var err error
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		var as []aggregate.Aggregate
		estr := estream.InMemory(events...)
		b.StartTimer()

		str := stream.New(estr, opts...)
		for str.Next(context.Background()) {
			name, id := str.Current()
			a := &mockAggregate{Aggregate: aggregate.New(name, id)}
			as = append(as, a)
		}
		if err = str.Err(); err != nil {
			b.Fatalf("stream: %v", err)
		}
		if err = str.Close(context.Background()); err != nil {
			b.Fatalf("close stream: %v", err)
		}
	}
}

func (a *mockAggregate) ApplyEvent(evt event.Event) {
	for i, name := range names {
		if name != evt.Name() {
			continue
		}
		if i%2 == 0 {
			a.a += float64(i)
			a.a *= 1.5
			continue
		}
		a.b += a.b
	}
}

func makeAggregates(n int) []aggregate.Aggregate {
	as := make([]aggregate.Aggregate, n)
	for i := range as {
		name := randomName()
		as[i] = aggregate.New(name, uuid.New())
	}
	return as
}

func makeEvents(n int, as []aggregate.Aggregate, grouped, sorted bool) []event.Event {
	rand.Seed(time.Now().UnixNano())
	eventm := make(map[aggregate.Aggregate][]event.Event)
	for _, a := range as {
		events := make([]event.Event, n)
		for i := range events {
			evt := event.New(
				randomName(),
				test.FooEventData{},
				event.Aggregate(a.AggregateName(), a.AggregateID(), a.AggregateVersion()+i+1),
			)
			events[i] = evt
		}
		if !sorted {
			rand.Shuffle(len(events), func(i, j int) {
				events[i], events[j] = events[j], events[i]
			})
		}
		eventm[a] = events
	}
	out := make([]event.Event, 0, len(as)*n)
	for _, events := range eventm {
		out = append(out, events...)
	}
	if !grouped {
		rand.Shuffle(len(out), func(i, j int) {
			out[i], out[j] = out[j], out[i]
		})
	}
	return out
}

func randomName() string {
	return names[rand.Intn(7)]
}