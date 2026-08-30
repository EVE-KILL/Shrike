package zkb

import (
	"context"
	"sync"
	"testing"
	"time"
)

// The limiter is the one piece of this package that protects zKillboard rather
// than us, so it is tested on a fake clock: real timing would make the tests
// both slow and flaky, and neither would actually prove the bound holds.

// fakeClock advances only when something sleeps on it, which makes the pacing
// deterministic and the test instant.
type fakeClock struct {
	mu    sync.Mutex
	now   time.Time
	slept []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Unix(1_700_000_000, 0)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Sleep(_ context.Context, d time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.slept = append(c.slept, d)
	c.now = c.now.Add(d)
	return nil
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func (c *fakeClock) sleeps() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.slept...)
}

func testLimiter(limit int, window time.Duration) (*WindowLimiter, *fakeClock) {
	clock := newFakeClock()
	l := NewWindowLimiter(limit, window)
	l.now = clock.Now
	l.sleep = clock.Sleep
	return l, clock
}

// The first `limit` requests go straight through; that is what makes the feed
// fast enough to keep up.
func TestLimiterAllowsTheFullBurst(t *testing.T) {
	l, clock := testLimiter(10, time.Second)
	ctx := context.Background()

	for i := range 10 {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
	}
	if got := clock.sleeps(); len(got) != 0 {
		t.Errorf("the first 10 requests slept %v, want none", got)
	}
}

// The eleventh must wait, and must wait exactly until the oldest request leaves
// the window rather than for a fixed guess.
func TestLimiterBlocksTheRequestOverTheLimit(t *testing.T) {
	l, clock := testLimiter(10, time.Second)
	ctx := context.Background()

	for range 10 {
		if err := l.Wait(ctx); err != nil {
			t.Fatal(err)
		}
	}

	// 400ms into the window, so the oldest entry ages out 600ms from now.
	clock.advance(400 * time.Millisecond)

	if err := l.Wait(ctx); err != nil {
		t.Fatal(err)
	}

	got := clock.sleeps()
	if len(got) != 1 {
		t.Fatalf("the 11th request slept %d times, want exactly 1: %v", len(got), got)
	}
	if got[0] != 600*time.Millisecond {
		t.Errorf("slept %v, want 600ms — the wait must be until the oldest request "+
			"leaves the window, not a fixed interval", got[0])
	}
}

// This is the assertion that actually protects zKillboard: over any window, no
// more than `limit` requests are permitted. A limiter that admits everything
// would pass every other test in this file.
//
// The window checked is half-open — (at-window, at] — which is the semantics
// the limiter implements and the TypeScript client implemented before it. The
// closed interval genuinely can hold 2×limit: ten requests land at t=0, the
// eleventh sleeps until exactly t=1s, and nine more follow it immediately, so
// [0s, 1s] contains twenty. That is inherent to any sliding window that expires
// entries at exactly `window` and is why the limit here is half of what
// zKillboard actually permits — the worst-case instantaneous burst is 20, which
// is their documented ceiling rather than an overrun of it.
func TestLimiterNeverExceedsTheRateOverTime(t *testing.T) {
	const limit = 10
	l, clock := testLimiter(limit, time.Second)
	ctx := context.Background()

	var stamps []time.Time
	for range 55 {
		if err := l.Wait(ctx); err != nil {
			t.Fatal(err)
		}
		stamps = append(stamps, clock.Now())
	}

	// For every request, count how many others fall inside the window ending at
	// it. More than `limit` anywhere means the bound was broken.
	for i, at := range stamps {
		n := 0
		for _, other := range stamps {
			if other.After(at.Add(-time.Second)) && !other.After(at) {
				n++
			}
		}
		if n > limit {
			t.Fatalf("request %d had %d requests in the second ending at it, "+
				"want at most %d — the limiter is oversubscribing zKillboard", i, n, limit)
		}
	}

	// And the converse: it must not be pointlessly slow. 55 requests at 10/s
	// cannot finish in under 5 seconds, and should not take much more.
	elapsed := clock.Now().Sub(time.Unix(1_700_000_000, 0))
	if elapsed < 5*time.Second {
		t.Errorf("55 requests took %v, which is faster than 10/s allows", elapsed)
	}
	if elapsed > 6*time.Second {
		t.Errorf("55 requests took %v, want about 5s — the limiter is over-waiting", elapsed)
	}
}

// Old timestamps have to be dropped, or the limiter degrades into "limit
// requests, ever".
func TestLimiterForgetsRequestsOutsideTheWindow(t *testing.T) {
	l, clock := testLimiter(10, time.Second)
	ctx := context.Background()

	for range 10 {
		if err := l.Wait(ctx); err != nil {
			t.Fatal(err)
		}
	}

	// A full window passes with no traffic.
	clock.advance(1500 * time.Millisecond)

	for i := range 10 {
		if err := l.Wait(ctx); err != nil {
			t.Fatal(err)
		}
		if got := clock.sleeps(); len(got) != 0 {
			t.Fatalf("request %d after an idle window slept %v — the limiter is not "+
				"forgetting requests that have aged out", i, got)
		}
	}
}

// The listener runs one goroutine today, but the limiter is shared with the
// repair cron, so concurrent use must not corrupt the timestamp slice.
func TestLimiterIsSafeUnderConcurrentUse(t *testing.T) {
	l := NewWindowLimiter(50, 10*time.Millisecond)
	ctx := context.Background()

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 25 {
				if err := l.Wait(ctx); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	wg.Wait()
}

// A cancelled context must abandon the wait rather than hold a shutdown open
// for the rest of the window.
func TestLimiterRespectsCancellation(t *testing.T) {
	l := NewWindowLimiter(1, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())

	if err := l.Wait(ctx); err != nil {
		t.Fatal(err)
	}

	cancel()
	if err := l.Wait(ctx); err == nil {
		t.Error("Wait ignored a cancelled context and would block for an hour")
	}
}
