package jobs

import (
	"testing"
	"time"
)

// ParseSchedule cannot use time.ParseDuration, which rejects the "d" unit that
// twelve of the crons rely on. These pin the units and the rejection cases.
func TestParseSchedule(t *testing.T) {
	ok := []struct {
		in   string
		want time.Duration
	}{
		{"1s", time.Second},
		{"30s", 30 * time.Second},
		{"1m", time.Minute},
		{"10m", 10 * time.Minute},
		{"30m", 30 * time.Minute},
		{"1h", time.Hour},
		{"6h", 6 * time.Hour},
		{"12h", 12 * time.Hour},
		{"1d", 24 * time.Hour},
	}
	for _, tc := range ok {
		got, err := ParseSchedule(tc.in)
		if err != nil {
			t.Errorf("ParseSchedule(%q) errored: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseSchedule(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}

	bad := []string{"", "s", "1", "1w", "1y", "-5m", "0h", "abc", "m1", "1.5h"}
	for _, in := range bad {
		if _, err := ParseSchedule(in); err == nil {
			t.Errorf("ParseSchedule(%q) succeeded; want an error", in)
		}
	}
}

// Every declared schedule must parse. A typo here would mean a job that silently
// never runs, which is exactly the class of bug this registry exists to prevent.
func TestEveryCronScheduleParses(t *testing.T) {
	for _, c := range Crons {
		d, err := ParseSchedule(c.Schedule)
		if err != nil {
			t.Errorf("cron %q has unparseable schedule %q: %v", c.Name, c.Schedule, err)
			continue
		}
		if d <= 0 {
			t.Errorf("cron %q resolved to non-positive interval %v", c.Name, d)
		}
	}
}

// Names become River job kinds and Redis keys, so duplicates or stray
// non-snake_case spellings must not creep back in.
func TestRegistryNamesAreUniqueAndNormalised(t *testing.T) {
	seen := map[string]string{}

	check := func(kind, name string) {
		if prev, dup := seen[name]; dup {
			t.Errorf("name %q declared twice (%s and %s)", name, prev, kind)
		}
		seen[name] = kind
		for _, r := range name {
			isLower := r >= 'a' && r <= 'z'
			isDigit := r >= '0' && r <= '9'
			if !isLower && !isDigit && r != '_' {
				t.Errorf("%s %q contains %q; names must be snake_case", kind, name, string(r))
			}
		}
	}

	for _, q := range Queues {
		check("queue", q.Name)
	}
	// Crons share the namespace check but not the map, since a queue and a cron
	// may legitimately share a name (battle_detection and campaign_processing do).
	seen = map[string]string{}
	for _, c := range Crons {
		check("cron", c.Name)
	}
}

// Legacy spellings must stay resolvable so existing invocations keep working.
func TestCronByNameResolvesLegacySpelling(t *testing.T) {
	c := CronByName("system-activity")
	if c == nil {
		t.Fatal("CronByName(\"system-activity\") = nil; the legacy name must still resolve")
	}
	if c.Name != "system_activity" {
		t.Errorf("resolved to %q; want the normalised system_activity", c.Name)
	}
	if CronByName("system_activity") == nil {
		t.Error("canonical name does not resolve")
	}
	if CronByName("no_such_cron") != nil {
		t.Error("unknown name resolved to something")
	}
}

func TestQueueByName(t *testing.T) {
	q := QueueByName("killmails")
	if q == nil {
		t.Fatal("QueueByName(\"killmails\") = nil")
	}
	// Spot-check a transcribed value against backend/src/queues/killmails/queue.ts.
	if q.Concurrency != 5 || q.Retries != 3 || !q.RequiresTQ {
		t.Errorf("killmails = %+v; want concurrency 5, retries 3, requiresTQ true", *q)
	}
	if QueueByName("nope") != nil {
		t.Error("unknown queue resolved to something")
	}
}

// Pending counts outstanding work only; completed and failed are history and
// must not inflate it.
func TestDepthPending(t *testing.T) {
	d := Depth{Waiting: 3, Prioritized: 5, Delayed: 2, Active: 7, Failed: 11, Completed: 13}
	if got := d.Pending(); got != 10 {
		t.Errorf("Pending() = %d; want 10 (3+5+2)", got)
	}
}
