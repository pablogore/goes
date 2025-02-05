package storetest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	stdtime "time"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/aggregate/snapshot"
	"github.com/modernice/goes/aggregate/snapshot/query"
	"github.com/modernice/goes/event/query/time"
	"github.com/modernice/goes/event/query/version"
	"github.com/modernice/goes/helper/pick"
	"github.com/modernice/goes/helper/streams"
	"github.com/modernice/goes/internal/xaggregate"
	"github.com/modernice/goes/internal/xtime"
)

// StoreFactory creates Stores.
type StoreFactory func() snapshot.Store

// Run runs the Store tests.
func Run(t *testing.T, newStore StoreFactory) {
	run(t, "Save", testSave, newStore)
	run(t, "Latest", testLatest, newStore)
	run(t, "Latest (multiple available)", testLatestMultipleAvailable, newStore)
	run(t, "Latest (not found)", testLatestNotFound, newStore)
	run(t, "Version", testVersion, newStore)
	run(t, "Version (not found)", testVersionNotFound, newStore)
	run(t, "Limit", testLimit, newStore)
	run(t, "Query", testQuery, newStore)
	run(t, "Delete", testDelete, newStore)
}

func run(t *testing.T, name string, runner func(*testing.T, StoreFactory), newStore StoreFactory) {
	t.Run(name, func(t *testing.T) {
		runner(t, newStore)
	})
}

func testSave(t *testing.T, newStore StoreFactory) {
	s := newStore()
	a := &snapshotter{
		Base:  aggregate.New("foo", uuid.New()),
		state: state{Foo: 3},
	}

	snap, err := snapshot.New(a)
	if err != nil {
		t.Fatalf("Marshal shouldn't fail; failed with %q", err)
	}

	if err := s.Save(context.Background(), snap); err != nil {
		t.Errorf("Save shouldn't fail; failed with %q", err)
	}
}

func testLatest(t *testing.T, newStore StoreFactory) {
	s := newStore()
	a := &snapshotter{
		Base:  aggregate.New("foo", uuid.New()),
		state: state{Foo: 3},
	}

	snap, err := snapshot.New(a)
	if err != nil {
		t.Fatalf("Marshal shouldn't fail; failed with %q", err)
	}

	if err := s.Save(context.Background(), snap); err != nil {
		t.Errorf("Save shouldn't fail; failed with %q", err)
	}

	latest, err := s.Latest(context.Background(), a.AggregateName(), a.AggregateID())
	if err != nil {
		t.Fatalf("Latest shouldn't fail; failed with %q", err)
	}

	if snap.AggregateName() != latest.AggregateName() {
		t.Errorf("AggregateName should return %q; got %q", snap.AggregateName(), latest.AggregateName())
	}

	if snap.AggregateID() != latest.AggregateID() {
		t.Errorf("AggregateID should return %q; got %q", snap.AggregateID(), latest.AggregateID())
	}

	if snap.AggregateVersion() != latest.AggregateVersion() {
		t.Errorf("AggregateVersion should return %q; got %q", snap.AggregateVersion(), latest.AggregateVersion())
	}

	wantTime := snap.Time()
	if !latest.Time().Equal(wantTime) {
		t.Errorf("Time should return %v; got %v", wantTime, latest.Time())
	}

	if !bytes.Equal(snap.State(), latest.State()) {
		t.Errorf("Data should return %v; got %v", snap.State(), latest.State())
	}
}

func testLatestMultipleAvailable(t *testing.T, newStore StoreFactory) {
	s := newStore()
	id := uuid.New()
	a10 := &snapshotter{Base: aggregate.New("foo", id, aggregate.Version(10))}
	a20 := &snapshotter{Base: aggregate.New("foo", id, aggregate.Version(20))}
	snap10, _ := snapshot.New(a10)
	snap20, _ := snapshot.New(a20)

	if err := s.Save(context.Background(), snap20); err != nil {
		t.Errorf("Save shouldn't fail; failed with %q", err)
	}

	if err := s.Save(context.Background(), snap10); err != nil {
		t.Errorf("Save shouldn't fail; failed with %q", err)
	}

	latest, err := s.Latest(context.Background(), "foo", id)
	if err != nil {
		t.Fatalf("Latest shouldn't fail; failed with %q", err)
	}

	if latest.AggregateName() != "foo" {
		t.Errorf("AggregateName should return %q; got %q", "foo", latest.AggregateName())
	}

	if latest.AggregateID() != id {
		t.Errorf("AggregateID should return %q; got %q", id, latest.AggregateID())
	}

	if latest.AggregateVersion() != 20 {
		t.Errorf("AggregateVersion should return %q; got %q", 20, latest.AggregateVersion())
	}

	wantTime := snap20.Time()
	if !latest.Time().Equal(wantTime) {
		t.Errorf("Time should return %v; got %v", wantTime, latest.Time())
	}

	if !bytes.Equal(snap20.State(), latest.State()) {
		t.Errorf("Data should return %v; got %v", snap20.State(), latest.State())
	}
}

func testLatestNotFound(t *testing.T, newStore StoreFactory) {
	s := newStore()
	snap, err := s.Latest(context.Background(), "foo", uuid.New())
	if snap != nil {
		t.Errorf("Latest should return no Snapshot; got %v", snap)
	}

	if err == nil {
		t.Errorf("Latest should fail; got %q", err)
	}
}

func testVersion(t *testing.T, newStore StoreFactory) {
	s := newStore()
	id := uuid.New()
	a10 := &snapshotter{Base: aggregate.New("foo", id, aggregate.Version(10))}
	a20 := &snapshotter{Base: aggregate.New("foo", id, aggregate.Version(20))}
	snap10, _ := snapshot.New(a10)
	snap20, _ := snapshot.New(a20)

	if err := s.Save(context.Background(), snap10); err != nil {
		t.Fatalf("failed to save Snapshot: %v", err)
	}

	if err := s.Save(context.Background(), snap20); err != nil {
		t.Fatalf("failed to save Snapshot: %v", err)
	}

	snap, err := s.Version(context.Background(), "foo", id, 10)
	if err != nil {
		t.Fatalf("Version shouldn't fail; failed with %q", err)
	}

	if snap.AggregateVersion() != snap10.AggregateVersion() {
		t.Errorf(
			"Version should return Snapshot with version %d; got version %d",
			snap10.AggregateVersion(),
			snap.AggregateVersion(),
		)
	}
}

func testVersionNotFound(t *testing.T, newStore StoreFactory) {
	s := newStore()

	snap, err := s.Version(context.Background(), "foo", uuid.New(), 10)
	if snap != nil {
		t.Errorf("Version should return no Snapshot; got %v", snap)
	}

	if err == nil {
		t.Errorf("Version should fail; got %q", err)
	}
}

func testLimit(t *testing.T, newStore StoreFactory) {
	run(t, "Basic", testLimitBasic, newStore)
	run(t, "NotFound", testLimitNotFound, newStore)
}

func testLimitBasic(t *testing.T, newStore StoreFactory) {
	s := newStore()

	id := uuid.New()
	as := []aggregate.Aggregate{
		&snapshotter{Base: aggregate.New("foo", id, aggregate.Version(1))},
		&snapshotter{Base: aggregate.New("foo", id, aggregate.Version(5))},
		&snapshotter{Base: aggregate.New("foo", id, aggregate.Version(10))},
		&snapshotter{Base: aggregate.New("foo", id, aggregate.Version(20))},
	}
	snaps := makeSnaps(as)

	for _, snap := range snaps {
		if err := s.Save(context.Background(), snap); err != nil {
			t.Fatalf("Save shouldn't fail; failed with %q", err)
		}
	}

	snap, err := s.Limit(context.Background(), "foo", id, 19)
	if err != nil {
		t.Fatalf("Limit shouldn't fail; failed with %q", err)
	}

	if snap.AggregateVersion() != 10 {
		t.Errorf("Limit should return the Snapshot with version %d; got version %d", 10, snap.AggregateVersion())
	}
}

func testLimitNotFound(t *testing.T, newStore StoreFactory) {
	s := newStore()

	id := uuid.New()
	as := []aggregate.Aggregate{
		&snapshotter{Base: aggregate.New("foo", id, aggregate.Version(10))},
		&snapshotter{Base: aggregate.New("foo", id, aggregate.Version(20))},
	}
	snaps := makeSnaps(as)

	for _, snap := range snaps {
		if err := s.Save(context.Background(), snap); err != nil {
			t.Fatalf("Save shouldn't fail; failed with %q", err)
		}
	}

	snap, err := s.Limit(context.Background(), "foo", id, 9)
	if err == nil {
		t.Errorf("Limit should fail!")
	}

	if snap != nil {
		t.Errorf("Limit should return no Snapshot; got %v", snap)
	}
}

func testQuery(t *testing.T, newStore StoreFactory) {
	run(t, "Name", testQueryName, newStore)
	run(t, "ID", testQueryID, newStore)
	run(t, "Version", testQueryVersion, newStore)
	run(t, "Time", testQueryTime, newStore)
	run(t, "Sorting", testQuerySorting, newStore)
}

func testQueryName(t *testing.T, newStore StoreFactory) {
	s := newStore()
	foos, _ := xaggregate.Make(5, xaggregate.Name("foo"))
	bars, _ := xaggregate.Make(5, xaggregate.Name("bar"))

	for i, foo := range foos {
		id, name, _ := foo.Aggregate()
		foos[i] = &snapshotter{Base: aggregate.New(name, id)}
	}

	for i, bar := range bars {
		id, name, _ := bar.Aggregate()
		bars[i] = &snapshotter{Base: aggregate.New(name, id)}
	}

	fooSnaps := makeSnaps(foos)
	barSnaps := makeSnaps(bars)
	snaps := append(fooSnaps, barSnaps...)

	for _, snap := range snaps {
		if err := s.Save(context.Background(), snap); err != nil {
			t.Fatalf("Save shouldn't fail; failed with %q", err)
		}
	}

	result, err := runQuery(s, query.New(query.Name("foo")))
	if err != nil {
		t.Fatal(err)
	}

	assertSame(t, fooSnaps, result)
}

func testQueryID(t *testing.T, newStore StoreFactory) {
	s := newStore()
	as, _ := xaggregate.Make(5, xaggregate.Name("foo"))

	for i, a := range as {
		id, name, _ := a.Aggregate()
		as[i] = &snapshotter{Base: aggregate.New(name, id)}
	}

	snaps := makeSnaps(as)

	for _, snap := range snaps {
		if err := s.Save(context.Background(), snap); err != nil {
			t.Fatalf("Save shouldn't fail; failed with %q", err)
		}
	}

	result, err := runQuery(s, query.New(query.ID(
		pick.AggregateID(as[0]),
		pick.AggregateID(as[4]),
	)))
	if err != nil {
		t.Fatal(err)
	}

	assertSame(t, []snapshot.Snapshot{
		snaps[0],
		snaps[4],
	}, result)
}

func testQueryVersion(t *testing.T, newStore StoreFactory) {
	s := newStore()
	as := []aggregate.Aggregate{
		&snapshotter{Base: aggregate.New("foo", uuid.New(), aggregate.Version(1))},
		&snapshotter{Base: aggregate.New("foo", uuid.New(), aggregate.Version(5))},
		&snapshotter{Base: aggregate.New("foo", uuid.New(), aggregate.Version(10))},
	}
	snaps := makeSnaps(as)

	for _, snap := range snaps {
		if err := s.Save(context.Background(), snap); err != nil {
			t.Fatalf("Save shouldn't fail; failed with %q", err)
		}
	}

	result, err := runQuery(s, query.New(
		query.Version(version.Exact(1, 10)),
	))
	if err != nil {
		t.Fatal(err)
	}

	assertSame(t, []snapshot.Snapshot{
		snaps[0],
		snaps[2],
	}, result)
}

func testQueryTime(t *testing.T, newStore StoreFactory) {
	s := newStore()
	as := []aggregate.Aggregate{
		&snapshotter{Base: aggregate.New("foo", uuid.New())},
		&snapshotter{Base: aggregate.New("foo", uuid.New())},
		&snapshotter{Base: aggregate.New("foo", uuid.New())},
	}
	snaps := make([]snapshot.Snapshot, len(as))
	for i := range as {
		var err error
		var opts []snapshot.Option
		if i == 2 {
			opts = append(opts, snapshot.Time(xtime.Now().Add(-stdtime.Minute)))
		}
		if snaps[i], err = snapshot.New(as[i], opts...); err != nil {
			t.Fatalf("failed to make Snapshot: %v", err)
		}
	}

	for _, snap := range snaps {
		if err := s.Save(context.Background(), snap); err != nil {
			t.Fatalf("Save shouldn't fail; failed with %q", err)
		}
	}

	result, err := runQuery(s, query.New(
		query.Time(time.After(xtime.Now().Add(-stdtime.Second))),
	))
	if err != nil {
		t.Fatal(err)
	}

	assertSame(t, snaps[:2], result)
}

func testQuerySorting(t *testing.T, newStore StoreFactory) {
	ids := make([]uuid.UUID, 9)
	for i := range ids {
		ids[i] = uuid.New()
	}
	sort.Slice(ids, func(a, b int) bool {
		return ids[a].String() < ids[b].String()
	})
	as := []aggregate.Aggregate{
		&snapshotter{Base: aggregate.New("bar1", ids[0], aggregate.Version(1))},
		&snapshotter{Base: aggregate.New("bar2", ids[1], aggregate.Version(2))},
		&snapshotter{Base: aggregate.New("bar3", ids[2], aggregate.Version(3))},
		&snapshotter{Base: aggregate.New("baz1", ids[3], aggregate.Version(4))},
		&snapshotter{Base: aggregate.New("baz2", ids[4], aggregate.Version(5))},
		&snapshotter{Base: aggregate.New("baz3", ids[5], aggregate.Version(6))},
		&snapshotter{Base: aggregate.New("foo1", ids[6], aggregate.Version(7))},
		&snapshotter{Base: aggregate.New("foo2", ids[7], aggregate.Version(8))},
		&snapshotter{Base: aggregate.New("foo3", ids[8], aggregate.Version(9))},
	}
	snaps := makeSnaps(as)

	tests := []struct {
		name string
		q    query.Query
		want []snapshot.Snapshot
	}{
		{
			name: "SortAggregateName(asc)",
			q:    query.New(query.SortBy(aggregate.SortName, aggregate.SortAsc)),
			want: snaps,
		},
		{
			name: "SortAggregateName(desc)",
			q:    query.New(query.SortBy(aggregate.SortName, aggregate.SortDesc)),
			want: []snapshot.Snapshot{
				snaps[8], snaps[7], snaps[6],
				snaps[5], snaps[4], snaps[3],
				snaps[2], snaps[1], snaps[0],
			},
		},
		{
			name: "SortAggregateID(asc)",
			q:    query.New(query.SortBy(aggregate.SortID, aggregate.SortAsc)),
			want: snaps,
		},
		{
			name: "SortAggregateID(desc)",
			q:    query.New(query.SortBy(aggregate.SortID, aggregate.SortDesc)),
			want: []snapshot.Snapshot{
				snaps[8], snaps[7], snaps[6],
				snaps[5], snaps[4], snaps[3],
				snaps[2], snaps[1], snaps[0],
			},
		},
		{
			name: "SortAggregateVersion(asc)",
			q:    query.New(query.SortBy(aggregate.SortVersion, aggregate.SortAsc)),
			want: snaps,
		},
		{
			name: "SortAggregateVersion(desc)",
			q:    query.New(query.SortBy(aggregate.SortVersion, aggregate.SortDesc)),
			want: []snapshot.Snapshot{
				snaps[8], snaps[7], snaps[6],
				snaps[5], snaps[4], snaps[3],
				snaps[2], snaps[1], snaps[0],
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStore()
			for _, snap := range snaps {
				if err := store.Save(context.Background(), snap); err != nil {
					t.Fatalf("Save shouldn't fail; failed with %q", err)
				}
			}

			result, err := runQuery(store, tt.q)
			if err != nil {
				t.Fatalf("query failed with %q", err)
			}

			assertEqual(t, tt.want, result)
		})
	}
}

func testDelete(t *testing.T, newStore StoreFactory) {
	s := newStore()
	a := &snapshotter{Base: aggregate.New("foo", uuid.New())}
	snap, _ := snapshot.New(a)

	if err := s.Save(context.Background(), snap); err != nil {
		t.Fatalf("Save shouldn't fail; failed with %q", err)
	}

	if err := s.Delete(context.Background(), snap); err != nil {
		t.Fatalf("Delete shouldn't fail; failed with %q", err)
	}

	snap, err := s.Latest(context.Background(), a.AggregateName(), a.AggregateID())
	if err == nil {
		t.Errorf("Latest should fail with an error; got %q", err)
	}
	if snap != nil {
		t.Errorf("Latest shouldn't return a Snapshot; got %v", snap)
	}
}

func runQuery(s snapshot.Store, q snapshot.Query) ([]snapshot.Snapshot, error) {
	str, errs, err := s.Query(context.Background(), q)
	if err != nil {
		return nil, fmt.Errorf("expected store.Query to succeed; got %w", err)
	}
	return streams.Drain(context.Background(), str, errs)
}

func makeSnaps(as []aggregate.Aggregate) []snapshot.Snapshot {
	snaps := make([]snapshot.Snapshot, len(as))
	for i, a := range as {
		snap, err := snapshot.New(a)
		if err != nil {
			panic(err)
		}
		snaps[i] = snap
	}
	return snaps
}

func assertEqual(t *testing.T, want, got []snapshot.Snapshot) {
	if len(want) != len(got) {
		t.Fatalf("(len(want) == %d) != (len(got) == %d)", len(want), len(got))
	}

	for i, snap := range want {
		gs := got[i]
		if snap.AggregateName() != gs.AggregateName() {
			t.Errorf(
				"want[%d].AggregateName() == %q; got[%d].AggregateName() == %q",
				i,
				snap.AggregateName(),
				i,
				gs.AggregateName(),
			)
		}
		if snap.AggregateID() != gs.AggregateID() {
			t.Errorf(
				"want[%d].AggregateID() == %s; got[%d].AggregateID() == %s",
				i,
				snap.AggregateID(),
				i,
				gs.AggregateID(),
			)
		}
		if snap.AggregateVersion() != gs.AggregateVersion() {
			t.Errorf(
				"want[%d].AggregateVersion() == %d; got[%d].AggregateVersion() == %d",
				i,
				snap.AggregateVersion(),
				i,
				gs.AggregateVersion(),
			)
		}
	}
}

func assertSame(t *testing.T, want, got []snapshot.Snapshot) {
	sort.Slice(want, func(a, b int) bool {
		return want[a].AggregateID().String() <= want[b].AggregateID().String()
	})
	sort.Slice(got, func(a, b int) bool {
		return got[a].AggregateID().String() <= got[b].AggregateID().String()
	})
	assertEqual(t, want, got)
}

type snapshotter struct {
	*aggregate.Base
	state state
}

type state struct {
	Foo int
}

func (ss *snapshotter) MarshalSnapshot() ([]byte, error) {
	return json.Marshal(ss.state)
}

func (ss *snapshotter) UnmarshalSnapshot(b []byte) error {
	return json.Unmarshal(b, &ss.state)
}
