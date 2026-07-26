package queue

import (
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/riverqueue/river/rivertype"
)

// Retry timing is the difference between riding out a five-minute ESI blip and
// hammering a struggling service until its error limit bans us. The formula
// matters, but so does the cap: without one the ESI queues retry from a
// 30-second base ten times, and 30s × 2⁹ is over four hours.

// fixedJitter removes the randomness so the formula can be asserted exactly.
// 0.5 is the midpoint, which the jitter maps to a factor of exactly 1.
func fixedPolicy() *RegistryRetryPolicy {
	return &RegistryRetryPolicy{Rand: func() float64 { return 0.5 }}
}

// The delay doubles per attempt from the queue's declared base, which is what
// BullMQ's exponential backoff did.
func TestBackoffDoublesFromTheDeclaredBase(t *testing.T) {
	p := fixedPolicy()

	// killmails declares a 1000ms base.
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 16 * time.Second},
	}
	for _, tc := range cases {
		if got := p.delay("killmails", tc.attempt); got != tc.want {
			t.Errorf("attempt %d waited %v, want %v", tc.attempt, got, tc.want)
		}
	}
}

// Each queue's own declared backoff is used, not one shared default. The ESI
// queues wait 30 seconds because ESI outages last minutes; the event queues
// wait 500ms because a websocket relay hiccup lasts milliseconds.
func TestBackoffUsesEachQueuesOwnBase(t *testing.T) {
	p := fixedPolicy()

	for _, q := range jobs.Queues {
		want := time.Duration(q.BackoffDelay) * time.Millisecond
		if got := p.delay(q.Name, 1); got != want {
			t.Errorf("%s: first retry waited %v, want its declared %v", q.Name, got, want)
		}
	}
}

// A kind with no registry entry still gets a sane delay rather than zero, which
// would retry instantly in a loop.
func TestBackoffFallsBackForAnUndeclaredKind(t *testing.T) {
	p := fixedPolicy()
	if got := p.delay("cron", 1); got != DefaultBackoff {
		t.Errorf("an undeclared kind waited %v, want %v", got, DefaultBackoff)
	}
}

// The cap is what stops a ten-retry queue from scheduling its last attempt four
// hours out, where it holds a backlog slot and helps nobody.
func TestBackoffIsCapped(t *testing.T) {
	p := fixedPolicy()

	// esi_character declares 30s and 10 retries; uncapped, attempt 10 would be
	// 30s × 2⁹ ≈ 4h16m.
	if got := p.delay("esi_character", 10); got != MaxRetryDelay {
		t.Errorf("the tenth retry waited %v, want it capped at %v", got, MaxRetryDelay)
	}

	// And nothing, at any attempt count, exceeds the cap by more than jitter.
	ceiling := float64(MaxRetryDelay) * (1 + RetryJitter)
	max := time.Duration(int64(ceiling))
	for _, q := range jobs.Queues {
		for attempt := 1; attempt <= 40; attempt++ {
			got := (&RegistryRetryPolicy{Rand: func() float64 { return 1 }}).delay(q.Name, attempt)
			if got > max {
				t.Fatalf("%s attempt %d waited %v, which exceeds the cap plus jitter (%v)",
					q.Name, attempt, got, max)
			}
		}
	}
}

// A huge attempt count must not overflow the duration into a negative, which
// would schedule the retry in the past and spin.
func TestBackoffDoesNotOverflow(t *testing.T) {
	p := fixedPolicy()
	for _, attempt := range []int{0, -5, 62, 63, 64, 1000, 1 << 20} {
		got := p.delay("esi_character", attempt)
		if got <= 0 {
			t.Errorf("attempt %d produced a delay of %v — a non-positive delay retries "+
				"immediately and turns a failing job into a hot loop", attempt, got)
		}
		if got > MaxRetryDelay*2 {
			t.Errorf("attempt %d produced %v, far beyond the cap", attempt, got)
		}
	}
}

// Jitter has to actually vary, or the correlated-failure stampede it exists to
// prevent still happens.
func TestJitterSpreadsRetries(t *testing.T) {
	// Two policies at the extremes of the jitter range.
	low := (&RegistryRetryPolicy{Rand: func() float64 { return 0 }}).delay("killmails", 3)
	high := (&RegistryRetryPolicy{Rand: func() float64 { return 1 }}).delay("killmails", 3)

	if low == high {
		t.Fatal("jitter produced the same delay at both extremes — every job failing " +
			"in the same second would retry in the same second, repeatedly")
	}
	if low >= high {
		t.Errorf("jitter is inverted: low=%v high=%v", low, high)
	}

	// The spread must be the declared fraction, not an arbitrary one.
	base := 4 * time.Second // killmails, attempt 3
	wantLow := time.Duration(float64(base) * (1 - RetryJitter))
	wantHigh := time.Duration(float64(base) * (1 + RetryJitter))
	if low != wantLow || high != wantHigh {
		t.Errorf("jitter range = [%v, %v], want [%v, %v]", low, high, wantLow, wantHigh)
	}
}

// Jitter must never drive a delay to zero or below.
func TestJitterNeverProducesANonPositiveDelay(t *testing.T) {
	p := &RegistryRetryPolicy{Rand: func() float64 { return 0 }}
	for _, q := range jobs.Queues {
		for attempt := 1; attempt <= 12; attempt++ {
			if got := p.delay(q.Name, attempt); got <= 0 {
				t.Fatalf("%s attempt %d waited %v", q.Name, attempt, got)
			}
		}
	}
}

// NextRetry is measured from now, not from when the attempt started.
func TestNextRetryIsRelativeToNow(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 30, 0, time.UTC)
	p := &RegistryRetryPolicy{
		Rand: func() float64 { return 0.5 },
		Now:  func() time.Time { return now },
	}

	// The attempt started well before now, as a slow job's would.
	attemptedAt := now.Add(-30 * time.Second)
	job := &rivertype.JobRow{Kind: "killmails", Attempt: 1, AttemptedAt: &attemptedAt}

	got := p.NextRetry(job)
	want := now.Add(time.Second)
	if !got.Equal(want) {
		t.Errorf("NextRetry = %s, want %s", got, want)
	}
}

// The retry must never land in the past, whatever the job did.
//
// This is a regression test for a bug found on the first live run. Measuring
// the delay from when the attempt started means a job that runs for longer than
// its own backoff gets a retry time already behind it — the killmails queue has
// a one-second base and a killmail that takes a second to fail lands exactly
// there. River rejects a past retry time and silently falls back to its own
// default policy, so every per-queue backoff in the registry stops being used
// and nothing says so except a warning in the log.
func TestNextRetryIsNeverInThePast(t *testing.T) {
	p := &RegistryRetryPolicy{Rand: func() float64 { return 0 }} // minimum jitter

	for _, q := range jobs.Queues {
		for attempt := 1; attempt <= 12; attempt++ {
			// A job that ran for far longer than its backoff before failing.
			before := time.Now().UTC()
			attemptedAt := before.Add(-time.Hour)
			job := &rivertype.JobRow{Kind: q.Name, Attempt: attempt, AttemptedAt: &attemptedAt}

			got := p.NextRetry(job)
			if !got.After(before) {
				t.Fatalf("%s attempt %d scheduled its retry at %s, which is not after "+
					"now (%s) — River would discard the declared backoff and use its "+
					"own default instead", q.Name, attempt, got, before)
			}
		}
	}
}
