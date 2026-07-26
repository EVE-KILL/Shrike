package zkb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// The listener is a cursor and a set of failure policies, and the failure
// policies are the whole point: a feed reader that stops on the first bad
// response, or advances past a killmail it never handed off, fails quietly and
// is noticed days later as a hole in the killboard.
//
// None of this needs a database or the network. The Store is faked, the clock
// is faked, and the feed is an httptest server, so the whole file runs in
// milliseconds and asserts behaviour rather than timing.

// fakeStore records what the listener asked it to do.
type fakeStore struct {
	mu sync.Mutex

	cursor      int64
	cursorSaves []int64

	// stored is what Has reports as already present.
	stored map[int64]bool

	accepted []int64

	// failAccept makes Accept fail for a killmail id, as a poisoned entry or a
	// database outage would.
	failAccept map[int64]bool
	// failHas does the same for the existence check.
	failHas map[int64]bool
}

func newFakeStore(cursor int64) *fakeStore {
	return &fakeStore{
		cursor:     cursor,
		stored:     map[int64]bool{},
		failAccept: map[int64]bool{},
		failHas:    map[int64]bool{},
	}
}

func (s *fakeStore) Cursor(context.Context) (int64, error) { return s.cursor, nil }

func (s *fakeStore) SaveCursor(_ context.Context, sequence int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cursor = sequence
	s.cursorSaves = append(s.cursorSaves, sequence)
	return nil
}

func (s *fakeStore) Has(_ context.Context, id int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failHas[id] {
		return false, errors.New("database unavailable")
	}
	return s.stored[id], nil
}

func (s *fakeStore) Accept(_ context.Context, r *Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failAccept[r.KillmailID] {
		return fmt.Errorf("cannot enqueue killmail %d", r.KillmailID)
	}
	s.accepted = append(s.accepted, r.KillmailID)
	return nil
}

func (s *fakeStore) acceptedIDs() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.accepted...)
}

func (s *fakeStore) saves() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.cursorSaves...)
}

// scriptedFeed serves a fixed set of sequences and 404s everything else, which
// is exactly how the real feed behaves at its head.
type scriptedFeed struct {
	mu       sync.Mutex
	head     int64
	entries  map[int64]string
	statuses map[int64]int
	requests []int64
}

func newScriptedFeed(head int64) *scriptedFeed {
	return &scriptedFeed{head: head, entries: map[int64]string{}, statuses: map[int64]int{}}
}

// add registers a killmail at a sequence.
func (f *scriptedFeed) add(sequence, killmailID int64) *scriptedFeed {
	f.entries[sequence] = fmt.Sprintf(
		`{"killmail_id":%d,"hash":"hash-%d","sequence_id":%d,"zkb":{"hash":"hash-%d"},
          "esi":{"killmail_id":%d,"killmail_time":"2026-07-20T12:00:00Z","solar_system_id":30000142}}`,
		killmailID, killmailID, sequence, killmailID, killmailID)
	return f
}

// status makes a sequence answer with an HTTP status instead of a body.
func (f *scriptedFeed) status(sequence int64, code int) *scriptedFeed {
	f.statuses[sequence] = code
	return f
}

func (f *scriptedFeed) client(t *testing.T) *Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/sequence.json") {
			_, _ = fmt.Fprintf(w, `{"sequence":%d}`, f.head)
			return
		}

		name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), ".json")
		seq, err := strconv.ParseInt(name, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		f.mu.Lock()
		f.requests = append(f.requests, seq)
		code, hasCode := f.statuses[seq]
		body, hasBody := f.entries[seq]
		f.mu.Unlock()

		switch {
		case hasCode:
			w.WriteHeader(code)
		case hasBody:
			_, _ = io.WriteString(w, body)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	return &Client{BaseURL: srv.URL, UserAgent: "shrike-test/1.0", HTTP: srv.Client()}
}

func (f *scriptedFeed) requested() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int64(nil), f.requests...)
}

// runListener drives a listener until it has emitted stopAfter events, then
// cancels. Sleeps are instant, so this is a pure logic test.
func runListener(t *testing.T, l *Listener, stopAfter int) []Event {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var events []Event
	l.Sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	l.OnEvent = func(e Event) {
		events = append(events, e)
		if len(events) >= stopAfter {
			cancel()
		}
	}

	// A guard against a listener that never emits: without it a logic bug turns
	// into a hung test rather than a failing one.
	done := make(chan struct{})
	go func() {
		_, _ = l.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("the listener did not finish — it is not making progress")
	}
	return events
}

// With no stored cursor the listener starts at the head of the feed. Starting
// anywhere else would walk millions of expired sequences.
func TestListenerBootstrapsFromTheHead(t *testing.T) {
	feed := newScriptedFeed(5000).add(5001, 111)
	store := newFakeStore(0)

	l := &Listener{Client: feed.client(t), Store: store}
	runListener(t, l, 1)

	if got := feed.requested(); len(got) == 0 || got[0] != 5001 {
		t.Fatalf("first request was for %v, want sequence 5001 (head + 1)", got)
	}
	// The bootstrapped position is written immediately, so a crash before the
	// first periodic save does not re-bootstrap at a newer head and skip
	// everything in between.
	if saves := store.saves(); len(saves) == 0 || saves[0] != 5000 {
		t.Errorf("cursor saves = %v, want the bootstrapped head 5000 written first", saves)
	}
}

// A stored cursor is resumed from, and the head is never consulted — asking for
// it would skip whatever arrived while the process was down.
func TestListenerResumesFromTheStoredCursor(t *testing.T) {
	feed := newScriptedFeed(9999).add(101, 111)
	store := newFakeStore(100)

	l := &Listener{Client: feed.client(t), Store: store}
	runListener(t, l, 1)

	got := feed.requested()
	if len(got) == 0 || got[0] != 101 {
		t.Fatalf("first request was for %v, want sequence 101 — the stored cursor "+
			"was ignored and everything since the last run would be skipped", got)
	}
}

// The happy path: consecutive entries are accepted in order.
func TestListenerAcceptsNewKillmails(t *testing.T) {
	feed := newScriptedFeed(100).add(101, 111).add(102, 222).add(103, 333)
	store := newFakeStore(100)

	l := &Listener{Client: feed.client(t), Store: store}
	runListener(t, l, 3)

	want := []int64{111, 222, 333}
	got := store.acceptedIDs()
	if len(got) != len(want) {
		t.Fatalf("accepted %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("accepted[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

// R2Z2 re-publishes mails during backfeed storms and the same kill also arrives
// from the ESI backfill, so a duplicate is routine. It must advance the cursor
// without being handed on.
func TestListenerSkipsKillmailsAlreadyStored(t *testing.T) {
	feed := newScriptedFeed(100).add(101, 111).add(102, 222)
	store := newFakeStore(100)
	store.stored[111] = true

	l := &Listener{Client: feed.client(t), Store: store}
	events := runListener(t, l, 2)

	if got := store.acceptedIDs(); len(got) != 1 || got[0] != 222 {
		t.Errorf("accepted %v, want only the killmail that was not already stored", got)
	}
	if events[0].Kind != "repost" {
		t.Errorf("event for the stored killmail was %q, want \"repost\"", events[0].Kind)
	}
	if events[1].Kind != "new" {
		t.Errorf("event for the fresh killmail was %q, want \"new\"", events[1].Kind)
	}
}

// A 404 is the steady state of a caught-up consumer. The cursor must stay put.
func TestListenerWaitsAtTheHeadWithoutBurningAttempts(t *testing.T) {
	feed := newScriptedFeed(100) // nothing at 101 yet
	store := newFakeStore(100)

	l := &Listener{Client: feed.client(t), Store: store}
	events := runListener(t, l, 8)

	for i, e := range events {
		if e.Kind != "caught-up" {
			t.Fatalf("event %d was %q, want every event to be \"caught-up\" — "+
				"an unwritten sequence is not a failure", i, e.Kind)
		}
		if e.Sequence != 101 {
			t.Fatalf("event %d was for sequence %d, want 101 — the cursor advanced "+
				"past an entry the feed never served", i, e.Sequence)
		}
	}
}

// A transient failure is retried on the same sequence, and succeeds when the
// failure clears — nothing is lost.
func TestListenerRetriesTheSameSequence(t *testing.T) {
	feed := newScriptedFeed(100).add(101, 111)
	store := newFakeStore(100)
	store.failAccept[111] = true

	l := &Listener{Client: feed.client(t), Store: store}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l.Sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }

	var attempts int
	l.OnEvent = func(e Event) {
		if e.Kind == "error" {
			attempts++
			if attempts == 2 {
				// The outage clears mid-retry.
				store.mu.Lock()
				store.failAccept[111] = false
				store.mu.Unlock()
			}
		}
		if e.Kind == "new" {
			cancel()
		}
	}

	done := make(chan struct{})
	go func() { _, _ = l.Start(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("the listener never recovered from a transient failure")
	}

	if got := store.acceptedIDs(); len(got) != 1 || got[0] != 111 {
		t.Errorf("accepted %v, want the killmail to survive a transient failure", got)
	}
}

// A permanently unacceptable entry must never be skipped. Advancing to the
// next dense R2Z2 sequence would make this killmail permanently absent from
// the live ingest path.
func TestListenerDoesNotSkipAPoisonedEntry(t *testing.T) {
	feed := newScriptedFeed(100).add(101, 111).add(102, 222)
	store := newFakeStore(100)
	store.failAccept[111] = true // never clears

	l := &Listener{Client: feed.client(t), Store: store}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l.Sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }

	var attempts int
	l.OnEvent = func(e Event) {
		if e.Kind == "error" {
			attempts++
		}
		if attempts == 8 {
			cancel()
		}
	}

	done := make(chan struct{})
	go func() { _, _ = l.Start(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("the listener did not keep retrying the poisoned entry")
	}

	if attempts != 8 {
		t.Errorf("attempts = %d, want 8 retries of the same sequence", attempts)
	}
	if got := store.acceptedIDs(); len(got) != 0 {
		t.Errorf("accepted %v, want no later entry while sequence 101 is unresolved", got)
	}
	if got := l.Stats().Sequence; got != 100 {
		t.Errorf("cursor = %d, want 100 while sequence 101 is unresolved", got)
	}
}

// A throttle must be retried, not treated as a missing entry, or the listener
// keeps hammering at the pace that earned the 429.
func TestListenerBacksOffOnThrottle(t *testing.T) {
	feed := newScriptedFeed(100).status(101, http.StatusTooManyRequests)
	store := newFakeStore(100)

	l := &Listener{Client: feed.client(t), Store: store}

	var waits []time.Duration
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l.Sleep = func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)
		return ctx.Err()
	}
	var events int
	l.OnEvent = func(Event) {
		events++
		if events >= 2 {
			cancel()
		}
	}

	done := make(chan struct{})
	go func() { _, _ = l.Start(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("the listener hung on a throttled feed")
	}

	if len(waits) == 0 {
		t.Fatal("a 429 produced no pause at all")
	}
	if waits[0] != WaitOnThrottled {
		t.Errorf("waited %v after a 429, want %v — a throttle needs a longer pause "+
			"than a caught-up 404, or the listener keeps earning 429s", waits[0], WaitOnThrottled)
	}
}

// The cursor is written periodically rather than per killmail. Losing it costs
// a replay, which is harmless; writing it every time costs a round trip per
// killmail, which is not.
func TestListenerSavesTheCursorPeriodically(t *testing.T) {
	feed := newScriptedFeed(0)
	const start = 1000
	for i := int64(1); i <= 120; i++ {
		feed.add(start+i, 5000+i)
	}
	store := newFakeStore(start)

	l := &Listener{Client: feed.client(t), Store: store}
	runListener(t, l, 120)

	saves := store.saves()
	if len(saves) < 2 {
		t.Fatalf("cursor saves = %v, want at least the periodic ones", saves)
	}

	// The last save is the unconditional one on shutdown and lands wherever the
	// listener happened to stop; everything before it is periodic.
	periodic := saves[:len(saves)-1]
	for _, s := range periodic {
		if s%SaveInterval != 0 {
			t.Errorf("cursor saved at %d, which is not a multiple of SaveInterval (%d)",
				s, SaveInterval)
		}
	}
	// 120 killmails from 1000 crosses 1050 and 1100, and no other multiple.
	if len(periodic) != 2 {
		t.Errorf("saved the cursor %d times over 120 killmails (%v), want 2 periodic "+
			"saves — writing it more often is a round trip per killmail for no gain",
			len(periodic), saves)
	}
}

// On the way out the cursor is written unconditionally, so a restart does not
// replay the whole window since the last periodic save.
func TestListenerSavesTheCursorOnShutdown(t *testing.T) {
	feed := newScriptedFeed(0)
	const start = 1000
	for i := int64(1); i <= 3; i++ {
		feed.add(start+i, 5000+i)
	}
	store := newFakeStore(start)

	l := &Listener{Client: feed.client(t), Store: store}
	runListener(t, l, 3)

	// Three killmails is well short of SaveInterval, so the only save can be
	// the shutdown one.
	saves := store.saves()
	if len(saves) == 0 {
		t.Fatal("no cursor was written on shutdown — a restart would replay from " +
			"the last periodic save")
	}
	if last := saves[len(saves)-1]; last != start+3 {
		t.Errorf("final cursor = %d, want %d", last, start+3)
	}
}

// Cancellation almost always arrives while the listener is paused at the head
// of the feed, because that is where it spends nearly all of its time. The
// cursor must still be written on the way out.
//
// This is a regression test for a bug found on the first live run: the shutdown
// save sat at the top of the loop, but a cancellation during a pause returned
// straight out of step() and skipped it, so the cursor stayed at the last
// periodic write and the next start replayed up to SaveInterval killmails.
func TestListenerSavesTheCursorWhenCancelledWhileWaiting(t *testing.T) {
	feed := newScriptedFeed(0)
	const start = 1000
	// Three killmails, then nothing — so the listener catches up and waits.
	for i := int64(1); i <= 3; i++ {
		feed.add(start+i, 5000+i)
	}
	store := newFakeStore(start)

	l := &Listener{Client: feed.client(t), Store: store}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel from inside the pause, which is exactly where a SIGTERM lands.
	l.Sleep = func(ctx context.Context, _ time.Duration) error {
		cancel()
		return ctx.Err()
	}

	done := make(chan struct{})
	go func() { _, _ = l.Start(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("the listener did not stop")
	}

	saves := store.saves()
	if len(saves) == 0 {
		t.Fatal("cancelling during a wait wrote no cursor at all")
	}
	// Three killmails is well short of SaveInterval, so the only possible save
	// is the shutdown one — and it has to be the position actually reached.
	if last := saves[len(saves)-1]; last != start+3 {
		t.Errorf("final cursor = %d, want %d — the position reached was not saved, "+
			"so a restart replays everything since the last periodic write",
			last, start+3)
	}
}

// A failure to read what is already stored is not a licence to advance: without
// knowing, the listener cannot tell a duplicate from a kill it is about to
// lose.
func TestListenerDoesNotAdvancePastAnUnreadableCheck(t *testing.T) {
	feed := newScriptedFeed(100).add(101, 111)
	store := newFakeStore(100)
	store.failHas[111] = true

	l := &Listener{Client: feed.client(t), Store: store}
	events := runListener(t, l, 3)

	for i, e := range events {
		if e.Kind != "error" {
			t.Fatalf("event %d was %q, want an error for every attempt", i, e.Kind)
		}
		if e.Sequence != 101 {
			t.Fatalf("event %d moved to sequence %d while the existence check was "+
				"failing — a killmail would be silently skipped", i, e.Sequence)
		}
	}
	if got := store.acceptedIDs(); len(got) != 0 {
		t.Errorf("accepted %v despite not knowing whether they were already stored", got)
	}
}

// Cancellation is the only way the loop exits.
func TestListenerStopsOnCancellation(t *testing.T) {
	feed := newScriptedFeed(100).add(101, 111)
	store := newFakeStore(100)

	l := &Listener{Client: feed.client(t), Store: store}
	l.Sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	stats, err := l.Start(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Start returned %v, want context.Canceled", err)
	}
	if stats.Accepted != 0 {
		t.Errorf("accepted %d killmails after cancellation", stats.Accepted)
	}
}
