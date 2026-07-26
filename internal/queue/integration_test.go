package queue

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"
)

// These are the assertions that cannot be made without a database, because they
// are about what River actually does rather than about what we asked it to do.
// Deduplication in particular is the whole reason the args types carry
// `river:"unique"` tags, and a tag that does not work looks exactly like a tag
// that does until two workers process the same killmail.

// testArgs is a job kind no worker consumes, so these tests never race a real
// worker on a shared development database.
type testArgs struct {
	ID   int64  `json:"id" river:"unique"`
	Note string `json:"note"`
}

func (testArgs) Kind() string { return "shrike_test_job" }

func insertOpts(priority Priority) *river.InsertOpts {
	return &river.InsertOpts{
		Queue:      testQueue,
		Priority:   int(priority),
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

// insertOnly builds a client that can enqueue but not consume.
func insertOnly(t *testing.T) *Client {
	t.Helper()
	pool := testPool(t)
	clearTestJobs(t, pool)

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// The same job enqueued twice must collapse into one. This is what the
// TypeScript expressed as a `dedupe:` string per call site, and the ones that
// were forgotten are exactly the jobs that ran twice.
func TestDuplicateJobsCollapse(t *testing.T) {
	c := insertOnly(t)
	ctx := context.Background()

	first, err := c.Insert(ctx, testArgs{ID: 42}, insertOpts(Live))
	if err != nil {
		t.Fatal(err)
	}
	if first.UniqueSkippedAsDuplicate {
		t.Fatal("the first insert was treated as a duplicate")
	}

	second, err := c.Insert(ctx, testArgs{ID: 42}, insertOpts(Live))
	if err != nil {
		t.Fatal(err)
	}
	if !second.UniqueSkippedAsDuplicate {
		t.Error("the same job was enqueued twice — it would be processed twice, " +
			"double-counting every statistic it touches")
	}
	if second.Job.ID != first.Job.ID {
		t.Errorf("the duplicate got a new job id (%d vs %d)", second.Job.ID, first.Job.ID)
	}
}

// Only the tagged field participates in the uniqueness check. Without the tag
// River hashes the whole args blob, so the same killmail arriving once with an
// embedded ESI body and once without would be two jobs and the kill would be
// processed twice.
func TestUniquenessIgnoresUntaggedFields(t *testing.T) {
	c := insertOnly(t)
	ctx := context.Background()

	if _, err := c.Insert(ctx, testArgs{ID: 77, Note: "from the feed"}, insertOpts(Live)); err != nil {
		t.Fatal(err)
	}

	second, err := c.Insert(ctx, testArgs{ID: 77, Note: "from the repair cron"}, insertOpts(Live))
	if err != nil {
		t.Fatal(err)
	}
	if !second.UniqueSkippedAsDuplicate {
		t.Error("two jobs differing only in an untagged field were both enqueued — " +
			"the same work would be done twice whenever two sources deliver it")
	}
}

// Different identities must not collapse, or the deduplication is silently
// dropping work.
func TestDistinctJobsDoNotCollapse(t *testing.T) {
	c := insertOnly(t)
	ctx := context.Background()

	for _, id := range []int64{1, 2, 3} {
		res, err := c.Insert(ctx, testArgs{ID: id}, insertOpts(Live))
		if err != nil {
			t.Fatal(err)
		}
		if res.UniqueSkippedAsDuplicate {
			t.Errorf("job %d collapsed into a different job — work is being dropped", id)
		}
	}
}

// The real args types have to deduplicate too, which is a different assertion
// from the synthetic one above: it checks the tags are actually on the fields
// that identify the work.
func TestRealJobTypesDeduplicate(t *testing.T) {
	pool := testPool(t)
	clearTestJobs(t, pool)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}

	// Ids well above anything CCP has issued, so nothing here collides with a
	// real backlog on a shared development database.
	const id = 2_100_000_001

	cases := []struct {
		name  string
		first river.JobArgs
		again river.JobArgs
	}{
		{
			// The killmail case is the important one: the same kill arrives from
			// the feed with a body and from the repair cron without one.
			name:  "killmails",
			first: KillmailArgs{KillmailID: id, KillmailHash: "abc", Killmail: nil},
			again: KillmailArgs{KillmailID: id, KillmailHash: "abc", WarID: 5},
		},
		{
			name:  "esi_character",
			first: CharacterArgs{CharacterID: id},
			again: CharacterArgs{CharacterID: id},
		},
		{
			// Force is outside the uniqueness key, so a forced refresh collapses
			// against a pending ordinary one rather than queueing alongside it.
			name:  "esi_character_history",
			first: CharacterHistoryArgs{CharacterID: id},
			again: CharacterHistoryArgs{CharacterID: id, Force: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = $1`, tc.first.Kind())
			}()

			if _, err := c.Insert(ctx, tc.first, InsertOptsFor(tc.first.Kind(), Live)); err != nil {
				t.Fatal(err)
			}
			second, err := c.Insert(ctx, tc.again, InsertOptsFor(tc.again.Kind(), Live))
			if err != nil {
				t.Fatal(err)
			}
			if !second.UniqueSkippedAsDuplicate {
				t.Errorf("%s was enqueued twice for the same entity", tc.name)
			}
		})
	}
}

// The declared retry budget has to reach the stored job, or a queue configured
// for ten attempts quietly gets River's default of three.
func TestStoredJobCarriesTheDeclaredRetryBudget(t *testing.T) {
	pool := testPool(t)
	clearTestJobs(t, pool)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}

	const id = 2_100_000_002
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'esi_character'`)
	}()

	res, err := c.Insert(ctx, CharacterArgs{CharacterID: id}, InsertOptsFor("esi_character", Live))
	if err != nil {
		t.Fatal(err)
	}

	// esi_character declares 10 retries.
	if res.Job.MaxAttempts != 10 {
		t.Errorf("MaxAttempts = %d, want the declared 10 — the queue would give up "+
			"after River's default of 3 instead", res.Job.MaxAttempts)
	}
	if res.Job.Priority != int(Live) {
		t.Errorf("Priority = %d, want %d", res.Job.Priority, int(Live))
	}
}

// Retained completed rows are diagnostics, not a dispatch lock. A stable job
// id such as a token or entity refresh must be eligible again as soon as its
// previous run has finalised.
func TestCompletedJobDoesNotBlockRedispatch(t *testing.T) {
	pool := testPool(t)
	clearTestJobs(t, pool)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}

	const id = 2_100_000_003
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'esi_character'`)
	}()

	first, err := Dispatch(ctx, c, CharacterArgs{CharacterID: id}, Live)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
        UPDATE river_job
        SET state = 'completed', finalized_at = now()
        WHERE id = $1`, first.Job.ID); err != nil {
		t.Fatal(err)
	}

	second, err := Dispatch(ctx, c, CharacterArgs{CharacterID: id}, Live)
	if err != nil {
		t.Fatal(err)
	}
	if second.Duplicate {
		t.Fatal("completed job blocked a new dispatch")
	}
	if second.Job.ID == first.Job.ID {
		t.Fatalf("redispatch reused completed job %d", first.Job.ID)
	}
}

// A live discovery must pull a dormant duplicate forward rather than leaving
// user-visible work behind a bulk backfill.
func TestDuplicateDispatchPromotesAvailableJob(t *testing.T) {
	pool := testPool(t)
	clearTestJobs(t, pool)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}

	const id = 2_100_000_004
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'esi_character'`)
	}()

	first, err := Dispatch(ctx, c, CharacterArgs{CharacterID: id}, DormantBackfill)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Dispatch(ctx, c, CharacterArgs{CharacterID: id}, Immediate)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate {
		t.Fatal("urgent dispatch did not collapse against the pending job")
	}
	if second.Job.ID != first.Job.ID {
		t.Fatalf("promoted job id = %d, want existing %d", second.Job.ID, first.Job.ID)
	}

	var priority int
	if err := pool.QueryRow(ctx,
		`SELECT priority FROM river_job WHERE id = $1`, first.Job.ID).Scan(&priority); err != nil {
		t.Fatal(err)
	}
	if priority != int(Immediate) {
		t.Fatalf("stored priority = %d, want immediate %d", priority, Immediate)
	}
}

// A batch insert must produce the same per-job settings as inserting one at a
// time, or the bulk paths silently get different behaviour from the single ones.
func TestBatchInsertMatchesSingleInsert(t *testing.T) {
	pool := testPool(t)
	clearTestJobs(t, pool)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'esi_alliance'`)
	}()

	batch := []river.JobArgs{
		AllianceArgs{AllianceID: 2_100_000_010},
		AllianceArgs{AllianceID: 2_100_000_011},
		AllianceArgs{AllianceID: 2_100_000_012},
	}

	n, err := DispatchMany(ctx, c, batch, queuePriorityForTest())
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("inserted %d of 3", n)
	}

	var maxAttempts, priority int
	err = pool.QueryRow(ctx, `
        SELECT max_attempts, priority FROM river_job
        WHERE kind = 'esi_alliance' AND args->>'alliance_id' = '2100000010'`).
		Scan(&maxAttempts, &priority)
	if err != nil {
		t.Fatal(err)
	}

	// esi_alliance declares 10 retries.
	if maxAttempts != 10 {
		t.Errorf("batch-inserted job has MaxAttempts %d, want 10", maxAttempts)
	}
	if priority != int(DormantBackfill) {
		t.Errorf("batch-inserted job has priority %d, want %d", priority, int(DormantBackfill))
	}

	// Re-inserting the same batch must collapse entirely.
	again, err := DispatchMany(ctx, c, batch, queuePriorityForTest())
	if err != nil {
		t.Fatal(err)
	}
	if again != 0 {
		t.Errorf("re-inserting an identical batch enqueued %d new jobs", again)
	}
}

func TestBatchDuplicateDispatchPromotesAvailableJobs(t *testing.T) {
	pool := testPool(t)
	clearTestJobs(t, pool)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'esi_alliance'`)
	}()

	batch := []river.JobArgs{
		AllianceArgs{AllianceID: 2_100_000_020},
		AllianceArgs{AllianceID: 2_100_000_021},
		AllianceArgs{AllianceID: 2_100_000_022},
	}
	if n, err := DispatchMany(ctx, c, batch, DormantBackfill); err != nil || n != len(batch) {
		t.Fatalf("initial batch inserted %d: %v", n, err)
	}
	if n, err := DispatchMany(ctx, c, batch, Immediate); err != nil || n != 0 {
		t.Fatalf("duplicate batch inserted %d: %v", n, err)
	}

	var promoted int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM river_job
		WHERE kind = 'esi_alliance'
		  AND args->>'alliance_id' IN ('2100000020', '2100000021', '2100000022')
		  AND priority = $1`, int(Immediate)).Scan(&promoted); err != nil {
		t.Fatal(err)
	}
	if promoted != len(batch) {
		t.Errorf("promoted %d batch duplicates, want %d", promoted, len(batch))
	}
}

func queuePriorityForTest() Priority { return DormantBackfill }

// The gate has to actually pause the queue rows River reads, not merely record
// an intention in memory. A gate that no-ops leaves ESI queues running through
// a downtime and burns the shared error budget.
//
// This starts a real consuming client, because that is what creates the
// river_queue rows the pause updates — an insert-only client has none, and the
// gate would appear to work while doing nothing.
func TestTQGatePausesAndResumes(t *testing.T) {
	pool := testPool(t)
	rdb := testRedis(t)
	ctx := context.Background()

	const watched = "esi_character"
	c := consumingClient(t, pool, watched)
	gate := &TQGate{Client: c, Redis: rdb}

	if err := rdb.Set(ctx, TQStatusKey, TQOffline, 0).Err(); err != nil {
		t.Fatal(err)
	}
	gate.check(ctx)

	if !queuePaused(t, pool, watched) {
		t.Fatal("TQ went offline and the ESI queue is still running — every job it " +
			"fetches will fail against a dead ESI and spend the shared error budget")
	}

	if err := rdb.Set(ctx, TQStatusKey, "online", 0).Err(); err != nil {
		t.Fatal(err)
	}
	gate.check(ctx)

	if queuePaused(t, pool, watched) {
		t.Error("TQ came back online and the queue is still paused — it would stay " +
			"stopped until someone noticed")
	}
}

// The gate must re-apply on every tick, not only when the flag changes.
//
// A pause issued before River created the queue rows silently does nothing. A
// gate that cached "already paused" would never try again, and the queue would
// run through the whole downtime while the gate believed it was stopped.
func TestTQGateReappliesAfterAMissedPause(t *testing.T) {
	pool := testPool(t)
	rdb := testRedis(t)
	ctx := context.Background()

	const watched = "esi_alliance"
	c := consumingClient(t, pool, watched)
	gate := &TQGate{Client: c, Redis: rdb}

	if err := rdb.Set(ctx, TQStatusKey, TQOffline, 0).Err(); err != nil {
		t.Fatal(err)
	}
	gate.check(ctx)
	if !queuePaused(t, pool, watched) {
		t.Fatal("the queue was not paused at all")
	}

	// Something resumes the queue out of band — another pod, or an operator.
	// The flag still says offline, so the next tick must put it back.
	if err := c.QueueResume(ctx, watched, nil); err != nil {
		t.Fatal(err)
	}
	if queuePaused(t, pool, watched) {
		t.Fatal("the out-of-band resume did not take effect, so the test proves nothing")
	}

	gate.check(ctx)
	if !queuePaused(t, pool, watched) {
		t.Error("the gate did not re-apply the pause — it cached that it had already " +
			"paused, so a pause that never took effect would never be retried")
	}
}

// A queue that does not need ESI keeps running through a downtime. Pausing
// everything would stop killmail processing for reasons unrelated to killmails.
func TestTQGateLeavesNonESIQueuesAlone(t *testing.T) {
	pool := testPool(t)
	rdb := testRedis(t)
	ctx := context.Background()

	// stats_writer needs no ESI.
	const unwatched = "stats_writer"
	c := consumingClient(t, pool, unwatched)
	gate := &TQGate{Client: c, Redis: rdb}

	if err := rdb.Set(ctx, TQStatusKey, TQOffline, 0).Err(); err != nil {
		t.Fatal(err)
	}
	gate.check(ctx)

	if queuePaused(t, pool, unwatched) {
		t.Errorf("%s was paused for a Tranquility outage it does not depend on", unwatched)
	}
}

// Redis being unreachable must not read as "TQ is offline". A Redis outage
// would otherwise halt all killmail processing — the opposite of what an
// operator wants during an incident.
func TestRedisOutageDoesNotLookLikeADowntime(t *testing.T) {
	ctx := context.Background()

	// A client pointed at a closed port. MaxRetries is -1 because go-redis
	// otherwise redials five times before reporting the failure, which turns a
	// test about error handling into a two-second wait.
	dead := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 100 * time.Millisecond,
		MaxRetries:  -1,
	})
	defer dead.Close() //nolint:errcheck

	offline, err := TQIsOffline(ctx, dead)
	if err == nil {
		t.Error("an unreachable Redis reported no error")
	}
	if offline {
		t.Error("an unreachable Redis was read as Tranquility being offline, which " +
			"would stop every ESI queue during a Redis incident")
	}
}

// An unset flag means online. On a fresh Redis, before the first tq_status run,
// the key does not exist — and reading that as a downtime would stop the whole
// cluster on startup.
func TestMissingFlagMeansOnline(t *testing.T) {
	rdb := testRedis(t)
	ctx := context.Background()

	if err := rdb.Del(ctx, TQStatusKey).Err(); err != nil {
		t.Fatal(err)
	}

	offline, err := TQIsOffline(ctx, rdb)
	if err != nil {
		t.Fatal(err)
	}
	if offline {
		t.Error("an unset status flag was read as offline — a fresh deployment would " +
			"pause every ESI queue before the first tq_status run")
	}
}

func queuePaused(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var pausedAt *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT paused_at FROM river_queue WHERE name = $1`, name).Scan(&pausedAt); err != nil {
		t.Fatalf("read queue %s: %v", name, err)
	}
	return pausedAt != nil
}

// consumingClient starts a real River client for the named queues, which is
// what creates their river_queue rows, and stops it on cleanup.
func consumingClient(t *testing.T, pool *pgxpool.Pool, queues ...string) *Client {
	t.Helper()

	workers := river.NewWorkers()
	river.AddWorker(workers, &noopWorker{})

	c, err := New(Options{Pool: pool, Workers: workers, Queues: queues})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := c.Start(ctx); err != nil {
		cancel()
		t.Fatalf("start River: %v", err)
	}

	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		_ = c.Stop(stopCtx)
		cancel()

		// Leave the queues running, so a paused row does not leak into the next
		// test or into a developer's local stack.
		for _, q := range queues {
			_ = c.QueueResume(context.Background(), q, nil)
		}
	})
	return c
}

// noopWorker exists only so the client has something registered; River refuses
// to start a client that consumes queues with no workers at all.
type noopWorker struct {
	river.WorkerDefaults[testArgs]
}

func (*noopWorker) Work(context.Context, *river.Job[testArgs]) error { return nil }

// --- Cron scheduling ---
//
// A recurring job has a uniqueness requirement that runs opposite to an
// ordinary one's, and getting it backwards fails in two different ways. Too
// loose and a slow cron piles up on itself; too tight and it runs once and
// never again. Both are tested here because both are silent.

// A cron already queued or running must not be queued again. The TypeScript
// runner guarded this in memory; through the queue the guarantee holds across
// every scheduler in the cluster.
func TestCronDoesNotOverlapItself(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)
	}()
	_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)

	first, err := c.Insert(ctx, CronArgs{Name: "sovereignty"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.UniqueSkippedAsDuplicate {
		t.Fatal("the first scheduled run was treated as a duplicate")
	}

	second, err := c.Insert(ctx, CronArgs{Name: "sovereignty"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !second.UniqueSkippedAsDuplicate {
		t.Error("a second run was queued while the first was still pending — a cron " +
			"that overruns its interval would stack copies of itself")
	}
}

// Different crons must never block each other. They share one job kind, so the
// uniqueness has to key on the name in the args rather than on the kind.
func TestDifferentCronsDoNotBlockEachOther(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)
	}()
	_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)

	for _, name := range []string{"sovereignty", "insurance", "tq_status"} {
		res, err := c.Insert(ctx, CronArgs{Name: name}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.UniqueSkippedAsDuplicate {
			t.Errorf("%s collapsed into another cron — they share a job kind, so the "+
				"uniqueness must key on the name in the args", name)
		}
	}
}

// The one that would break everything quietly: a completed run must not block
// the next tick.
//
// River's default unique states include completed, which is correct for an
// ordinary job and catastrophic for a recurring one — the completed row lingers
// for the retention period, so every cron would run once and then stop for
// twenty-four hours while still looking perfectly healthy.
func TestACompletedCronDoesNotBlockTheNextRun(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)
	}()
	_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)

	first, err := c.Insert(ctx, CronArgs{Name: "analyze"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Mark it finished, as a worker would.
	if _, err := pool.Exec(ctx,
		`UPDATE river_job SET state = 'completed', finalized_at = now() WHERE id = $1`,
		first.Job.ID); err != nil {
		t.Fatal(err)
	}

	second, err := c.Insert(ctx, CronArgs{Name: "analyze"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.UniqueSkippedAsDuplicate {
		t.Fatal("the next tick was blocked by the previous completed run — with " +
			"River's default retention this cron would run once and then not again " +
			"for 24 hours, while reporting no error at all")
	}
}

// A run that failed permanently must not block the next tick either, or one bad
// night stops the job until someone clears the row by hand.
func TestADiscardedCronDoesNotBlockTheNextRun(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)
	}()
	_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)

	first, err := c.Insert(ctx, CronArgs{Name: "wars"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE river_job SET state = 'discarded', finalized_at = now() WHERE id = $1`,
		first.Job.ID); err != nil {
		t.Fatal(err)
	}

	second, err := c.Insert(ctx, CronArgs{Name: "wars"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.UniqueSkippedAsDuplicate {
		t.Error("a permanently failed run blocked the next tick — the cron would stay " +
			"stopped until someone deleted the row by hand")
	}
}

// Scheduled jobs go to the cron queue, not to a work queue, so a slow hourly
// import cannot occupy the workers keeping the killmail feed moving.
func TestCronJobsAreRoutedToTheCronQueue(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	c, err := New(Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)
	}()
	_, _ = pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'cron'`)

	res, err := c.Insert(ctx, CronArgs{Name: "insurance"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Job.Queue != CronQueue {
		t.Errorf("a scheduled job landed on queue %q, want %q", res.Job.Queue, CronQueue)
	}
}
