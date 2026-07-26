package cron

import (
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
)

// The schedule is the one piece of the cron system whose behaviour is visible
// to anyone looking at the site: it decides that the daily stats rebuild
// happens at midnight rather than at 04:37 because that is when the pod last
// restarted. Getting it wrong is not a crash, it is a job that drifts.

// Boundaries are measured from the Unix epoch in UTC, so an hourly job fires at
// the top of the hour whatever time the process started.
func TestNextLandsOnWallClockBoundaries(t *testing.T) {
	cases := []struct {
		name     string
		interval time.Duration
		current  string
		want     string
	}{
		{"hourly from mid-hour", time.Hour, "2026-07-20T14:37:12Z", "2026-07-20T15:00:00Z"},
		{"hourly from the top", time.Hour, "2026-07-20T14:00:00Z", "2026-07-20T15:00:00Z"},
		{"daily from mid-day", 24 * time.Hour, "2026-07-20T14:37:12Z", "2026-07-21T00:00:00Z"},
		{"daily from midnight", 24 * time.Hour, "2026-07-20T00:00:00Z", "2026-07-21T00:00:00Z"},
		{"five minutes", 5 * time.Minute, "2026-07-20T14:37:12Z", "2026-07-20T14:40:00Z"},
		{"thirty seconds", 30 * time.Second, "2026-07-20T14:37:12Z", "2026-07-20T14:37:30Z"},
		{"six hours", 6 * time.Hour, "2026-07-20T14:37:12Z", "2026-07-20T18:00:00Z"},
		{"one minute", time.Minute, "2026-07-20T14:37:59.999Z", "2026-07-20T14:38:00Z"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			current, err := time.Parse(time.RFC3339Nano, tc.current)
			if err != nil {
				t.Fatal(err)
			}
			want, err := time.Parse(time.RFC3339Nano, tc.want)
			if err != nil {
				t.Fatal(err)
			}

			got := Every(tc.interval).Next(current)
			if !got.Equal(want) {
				t.Errorf("Next(%s) = %s, want %s", tc.current,
					got.Format(time.RFC3339Nano), tc.want)
			}
		})
	}
}

// Called at exactly a boundary — which is exactly when it is called, having
// just fired — the next fire must be the following boundary. Returning the same
// instant would run the job in a tight loop forever.
func TestNextIsStrictlyAfterTheCurrentTime(t *testing.T) {
	for _, interval := range []time.Duration{
		time.Second, 30 * time.Second, time.Minute, time.Hour, 24 * time.Hour,
	} {
		// Start exactly on a boundary.
		at := time.UnixMilli((time.Now().UnixMilli() / interval.Milliseconds()) * interval.Milliseconds()).UTC()

		next := Every(interval).Next(at)
		if !next.After(at) {
			t.Errorf("interval %v: Next(%s) = %s, which is not after it — the job "+
				"would fire continuously", interval, at, next)
		}
		if got := next.Sub(at); got != interval {
			t.Errorf("interval %v: the next fire is %v away, want exactly one interval",
				interval, got)
		}
	}
}

// Two consecutive fires are always exactly one interval apart, so a job cannot
// drift no matter how many times it has run.
func TestScheduleDoesNotDrift(t *testing.T) {
	const interval = 5 * time.Minute
	s := Every(interval)

	at := time.Date(2026, 7, 20, 14, 37, 12, 0, time.UTC)
	prev := s.Next(at)

	for range 500 {
		next := s.Next(prev)
		if got := next.Sub(prev); got != interval {
			t.Fatalf("consecutive fires were %v apart, want %v — the schedule drifts", got, interval)
		}
		// Every fire must remain aligned to the boundary, not merely spaced.
		if next.UnixMilli()%interval.Milliseconds() != 0 {
			t.Fatalf("fire at %s is not on a %v boundary", next, interval)
		}
		prev = next
	}
}

// A job whose previous run overran must still fire on the next boundary rather
// than immediately, and must not try to catch up on everything it missed.
func TestNextSkipsMissedBoundaries(t *testing.T) {
	s := Every(time.Hour)

	// The last fire was at 12:00 and it is now 14:37 — two boundaries were
	// missed while the job was running or the process was down.
	got := s.Next(time.Date(2026, 7, 20, 14, 37, 0, 0, time.UTC))
	want := time.Date(2026, 7, 20, 15, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Next = %s, want %s — a job that fell behind must resume at the "+
			"next boundary, not replay the ones it missed", got, want)
	}
}

// Every declared schedule string has to produce a working schedule; a typo in
// the registry should fail here rather than at three in the morning.
func TestEveryDeclaredCronHasAValidSchedule(t *testing.T) {
	for _, c := range jobs.Crons {
		s, err := ScheduleFor(c)
		if err != nil {
			t.Errorf("cron %s has schedule %q: %v", c.Name, c.Schedule, err)
			continue
		}
		if s.Interval() <= 0 {
			t.Errorf("cron %s resolved to a non-positive interval %v", c.Name, s.Interval())
		}

		now := time.Now().UTC()
		if next := s.Next(now); !next.After(now) {
			t.Errorf("cron %s scheduled its next run at %s, which is not in the future",
				c.Name, next)
		}
	}
}

// A malformed interval cannot be scheduled, and must not silently become a
// busy loop.
func TestZeroIntervalDoesNotBusyLoop(t *testing.T) {
	now := time.Now().UTC()
	next := Every(0).Next(now)
	if !next.After(now) {
		t.Errorf("a zero interval scheduled its next run at %s, which is not in the future", next)
	}
}
