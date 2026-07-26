package queue

import (
	"testing"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/riverqueue/river"
)

// Priorities are the mechanism that keeps a million-character backfill from
// starving the fetch for someone who just appeared on a killmail. The failure
// is not an error — everything still runs — it is a killboard that shows
// "Unknown" for minutes while the queue works through unrelated repair work.

// River accepts 1 through 4 and nothing else. A tier outside that range is
// rejected at insert time, which would take a whole queue down.
func TestTiersAreValidRiverPriorities(t *testing.T) {
	tiers := []Priority{Immediate, Live, RecentBackfill, DormantBackfill}
	for _, p := range tiers {
		if !p.Valid() {
			t.Errorf("%s (%d) is outside River's 1–4 range", p, int(p))
		}
	}

	// And they must be distinct and ordered, or the tiers collapse into one.
	for i := 1; i < len(tiers); i++ {
		if tiers[i] <= tiers[i-1] {
			t.Errorf("%s (%d) does not sort after %s (%d)",
				tiers[i], int(tiers[i]), tiers[i-1], int(tiers[i-1]))
		}
	}
}

// The BullMQ numbering has to map onto the tiers unambiguously, because during
// the changeover both dispatchers are enqueuing the same work and their
// relative order has to be preserved.
func TestBullMQPrioritiesMapToTiers(t *testing.T) {
	cases := []struct {
		bull int
		want Priority
	}{
		{0, Immediate},
		{1, Live},
		{5, RecentBackfill},
		{10, DormantBackfill},

		// Values between the defined tiers round towards the more urgent one,
		// matching historyPriorityFromParent().
		{2, RecentBackfill},
		{4, RecentBackfill},
		{6, DormantBackfill},
		{100, DormantBackfill},

		// A negative priority is more urgent than immediate, which BullMQ
		// allows and nothing in the codebase uses. It must not wrap around to
		// the bottom tier.
		{-1, Immediate},
	}

	for _, tc := range cases {
		if got := FromBullMQ(tc.bull); got != tc.want {
			t.Errorf("FromBullMQ(%d) = %s, want %s", tc.bull, got, tc.want)
		}
	}
}

// Follow-up work inherits its parent's urgency. Without this a single dormant
// backfill job spawns immediate children and the tiers stop meaning anything —
// which is exactly how a backfill ends up ahead of the live feed.
func TestCascadeInheritsTheParentTier(t *testing.T) {
	for _, p := range []Priority{Immediate, Live, RecentBackfill, DormantBackfill} {
		if got := CascadePriority(p); got != p {
			t.Errorf("a %s job's follow-up work became %s", p, got)
		}
	}

	// An unset or nonsensical parent falls back to Live rather than Immediate:
	// treating silence as maximum urgency is how the immediate lane fills with
	// routine work.
	for _, bad := range []Priority{0, -1, 99} {
		if got := CascadePriority(bad); got != Live {
			t.Errorf("CascadePriority(%d) = %s, want live", int(bad), got)
		}
	}
}

// Insert options come from the registry, so the retry budget a queue was
// declared with is the one it actually gets.
func TestInsertOptsComeFromTheRegistry(t *testing.T) {
	for _, q := range jobs.Queues {
		opts := InsertOptsFor(q.Name, Live)

		if opts.Queue != q.Name {
			t.Errorf("%s: opts.Queue = %q", q.Name, opts.Queue)
		}
		if opts.MaxAttempts != q.Retries {
			t.Errorf("%s: MaxAttempts = %d, want the declared %d",
				q.Name, opts.MaxAttempts, q.Retries)
		}
		if !opts.UniqueOpts.ByArgs {
			t.Errorf("%s: deduplication is off, so a repeated dispatch would process "+
				"the same work twice", q.Name)
		}
		if opts.Priority != int(Live) {
			t.Errorf("%s: Priority = %d, want %d", q.Name, opts.Priority, int(Live))
		}
	}
}

// An out-of-range priority must be corrected rather than passed to River, which
// would reject the insert and lose the job.
func TestInsertOptsClampAnInvalidPriority(t *testing.T) {
	for _, bad := range []Priority{0, -3, 9} {
		opts := InsertOptsFor("killmails", bad)
		if opts.Priority < 1 || opts.Priority > 4 {
			t.Errorf("priority %d produced River priority %d, which River rejects",
				int(bad), opts.Priority)
		}
	}
}

// A queue with no registry entry — the cron queue — still gets usable options.
func TestInsertOptsForAnUndeclaredQueue(t *testing.T) {
	opts := InsertOptsFor(CronQueue, Live)
	if opts.Queue != CronQueue {
		t.Errorf("Queue = %q", opts.Queue)
	}
	if opts.Priority < 1 || opts.Priority > 4 {
		t.Errorf("Priority = %d", opts.Priority)
	}
}

// Every job args type must report the queue name it belongs to, or River routes
// it to a queue no worker is consuming and it sits forever.
func TestJobKindsMatchDeclaredQueues(t *testing.T) {
	args := []river.JobArgs{
		KillmailArgs{},
		CharacterArgs{},
		CorporationArgs{},
		AllianceArgs{},
		CharacterHistoryArgs{},
		CorporationHistoryArgs{},
	}

	for _, a := range args {
		if jobs.QueueByName(a.Kind()) == nil {
			t.Errorf("job kind %q is not a declared queue", a.Kind())
		}
	}

	// The cron kind is deliberately not a declared queue — it is its own thing.
	if got := (CronArgs{}).Kind(); got != "cron" {
		t.Errorf("CronArgs.Kind() = %q, want \"cron\"", got)
	}
	if (CronArgs{}).InsertOpts().Queue != CronQueue {
		t.Error("cron jobs are not routed to the cron queue")
	}
}

// The default selection must never consume a queue another pod owns: the
// Discord bot consumes discord_events, and a backend worker taking those jobs
// would swallow them and deliver nothing.
func TestDefaultQueueSelectionSkipsForeignConsumers(t *testing.T) {
	selected := selectQueues(nil)

	var foreign int
	for _, q := range jobs.Queues {
		if q.ConsumerElsewhere {
			foreign++
		}
	}
	if foreign == 0 {
		t.Skip("no queue is marked ConsumerElsewhere, so there is nothing to skip")
	}

	for _, q := range selected {
		if q.ConsumerElsewhere {
			t.Errorf("%s is consumed by another pod but was selected by default — "+
				"this process would race the real consumer", q.Name)
		}
	}
	if len(selected) != len(jobs.Queues)-foreign {
		t.Errorf("selected %d queues, want %d", len(selected), len(jobs.Queues)-foreign)
	}
}

// An explicit request is honoured even for a foreign queue: that is someone who
// means it, and refusing would make the escape hatch useless.
func TestExplicitSelectionHonoursAForeignQueue(t *testing.T) {
	var foreign string
	for _, q := range jobs.Queues {
		if q.ConsumerElsewhere {
			foreign = q.Name
			break
		}
	}
	if foreign == "" {
		t.Skip("no queue is marked ConsumerElsewhere")
	}

	selected := selectQueues([]string{foreign})
	if len(selected) != 1 || selected[0].Name != foreign {
		t.Errorf("explicitly asking for %s selected %v", foreign, selected)
	}
}

// An unknown queue name selects nothing rather than everything, which would be
// a typo silently starting the whole worker set.
func TestUnknownQueueSelectsNothing(t *testing.T) {
	if got := selectQueues([]string{"not_a_queue"}); len(got) != 0 {
		t.Errorf("an unknown queue name selected %v", got)
	}
}
