package workers

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// The killmail worker is where the two ingest paths converge, and the property
// worth testing is that they converge on the same thing: a kill arriving with
// an embedded ESI document must produce exactly the rows a fetched one would,
// and arriving twice must produce them once.
//
// This needs a real database with the SDE loaded, because the parser resolves
// ship groups and system regions against it. It skips when that is not
// available rather than asserting against a stub, which would prove only that
// the stub matches itself.

const workersTestDSN = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = workersTestDSN
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// testDeps builds a dependency bundle backed by the local stack.
func testDeps(t *testing.T) *Deps {
	t.Helper()

	pool := testPool(t)
	ctx := context.Background()

	cache, err := eve.Load(ctx, pool)
	if err != nil {
		t.Skipf("static data is not loaded: %v", err)
	}
	// A cache with no types means an empty SDE, and every parse would resolve
	// to nothing — a test that passed against it would be asserting nothing.
	if cache.CountsByName()["inv_types"] == 0 {
		t.Skip("the SDE is not imported into the test database")
	}

	return &Deps{
		Pool:   pool,
		Cache:  cache,
		Prices: eve.NewPrices(pool, cache),
	}
}

// loadCorpusKillmail reads one of the parser's corpus mails.
//
// Reusing that corpus rather than inventing a fixture keeps one set of real
// killmails in the repository, and means this worker is exercised on documents
// already known to round-trip through the parser.
func loadCorpusKillmail(t *testing.T, id string) *killmail.ESIKillmail {
	t.Helper()

	path := filepath.Join("..", "killmail", "testdata", "corpus", id+".json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("corpus killmail %s is unavailable: %v", id, err)
	}

	var km killmail.ESIKillmail
	if err := json.Unmarshal(body, &km); err != nil {
		t.Fatalf("decode corpus killmail %s: %v", id, err)
	}
	return &km
}

// testJob wraps args in a River job.
//
// river.Job embeds *rivertype.JobRow, so a job literal built without one
// panics the moment the worker reads job.Priority — which every worker does, to
// decide the tier its follow-up work inherits.
func testJob(args queue.KillmailArgs, priority queue.Priority) *river.Job[queue.KillmailArgs] {
	return &river.Job[queue.KillmailArgs]{
		JobRow: &rivertype.JobRow{
			ID:       1,
			Kind:     args.Kind(),
			Queue:    args.Kind(),
			Priority: int(priority),
			Attempt:  1,
		},
		Args: args,
	}
}

// removeKillmail clears a killmail and everything hanging off it, before and
// after, so the test is repeatable on a database that already holds real data.
func removeKillmail(t *testing.T, pool *pgxpool.Pool, id int64) {
	t.Helper()

	clean := func() {
		_ = killmail.Delete(context.Background(), pool, id)
	}
	clean()
	t.Cleanup(clean)
}

// A killmail arriving with its ESI document embedded must be stored without any
// ESI request. This is the R2Z2 path, and it is the whole reason that feed is
// cheaper than the fetcher.
func TestWorkerStoresAnEmbeddedKillmail(t *testing.T) {
	d := testDeps(t)
	ctx := context.Background()

	km := loadCorpusKillmail(t, "103735737")
	removeKillmail(t, d.Pool, km.KillmailID)

	// A nil ESI client is the assertion: any attempt to fetch would panic, so
	// this passing proves the embedded document was used.
	d.ESI = nil

	w := &KillmailWorker{Deps: d}
	err := w.Work(ctx, testJob(queue.KillmailArgs{
		KillmailID:   km.KillmailID,
		KillmailHash: km.KillmailHash,
		Killmail:     km,
	}, queue.Live))
	if err != nil {
		t.Fatal(err)
	}

	stored, err := killmail.Load(ctx, d.Pool, km.KillmailID)
	if err != nil {
		t.Fatalf("the killmail was not stored: %v", err)
	}

	if stored.Killmail.KillmailHash != km.KillmailHash {
		t.Errorf("stored hash = %q, want %q", stored.Killmail.KillmailHash, km.KillmailHash)
	}
	if len(stored.Attackers) != len(km.Attackers) {
		t.Errorf("stored %d attackers, want %d", len(stored.Attackers), len(km.Attackers))
	}
	// A killmail valued at zero means the price lookup silently found nothing,
	// which is the failure that produces a killboard full of 0.00 ISK losses.
	if stored.Killmail.TotalValue <= 0 {
		t.Errorf("total_value = %v, want a positive valuation", stored.Killmail.TotalValue)
	}
}

// The same killmail twice must be stored once. Both feeds re-deliver routinely,
// so this is the common case rather than an edge one, and processing it twice
// double-counts every statistic derived from it.
func TestWorkerIsIdempotent(t *testing.T) {
	d := testDeps(t)
	ctx := context.Background()

	km := loadCorpusKillmail(t, "103735737")
	removeKillmail(t, d.Pool, km.KillmailID)
	d.ESI = nil

	w := &KillmailWorker{Deps: d}
	args := queue.KillmailArgs{
		KillmailID:   km.KillmailID,
		KillmailHash: km.KillmailHash,
		Killmail:     km,
	}

	for i := range 3 {
		if err := w.Work(ctx, testJob(args, queue.Live)); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	var killmails, attackers, items int64
	err := d.Pool.QueryRow(ctx, `
        SELECT (SELECT count(*) FROM killmails WHERE killmail_id = $1),
               (SELECT count(*) FROM killmail_attackers WHERE killmail_id = $1),
               (SELECT count(*) FROM killmail_items WHERE killmail_id = $1)`,
		km.KillmailID).Scan(&killmails, &attackers, &items)
	if err != nil {
		t.Fatal(err)
	}

	if killmails != 1 {
		t.Errorf("%d killmail rows after three deliveries, want 1", killmails)
	}
	if want := int64(len(km.Attackers)); attackers != want {
		t.Errorf("%d attacker rows after three deliveries, want %d", attackers, want)
	}
	if items == 0 {
		t.Error("no item rows were stored")
	}
}

// A killmail with no hash cannot be fetched and never will be. Retrying it
// three times asks an unanswerable question three times; cancelling records it
// once and moves on.
func TestWorkerCancelsAKillmailWithNoHash(t *testing.T) {
	d := testDeps(t)
	ctx := context.Background()

	const id = 2_100_000_099
	removeKillmail(t, d.Pool, id)

	w := &KillmailWorker{Deps: d}
	// No hash and no body: unanswerable.
	err := w.Work(ctx, testJob(queue.KillmailArgs{KillmailID: id}, queue.Live))

	if err == nil {
		t.Fatal("a killmail with neither a hash nor a body was accepted")
	}

	var cancel *river.JobCancelError
	if !errors.As(err, &cancel) {
		t.Errorf("returned %v (%T), want a JobCancel — an unanswerable request "+
			"should not consume the retry budget", err, err)
	}
}

// An already-stored killmail short-circuits before any parse or fetch. That is
// what keeps a backfeed storm cheap.
func TestWorkerSkipsAStoredKillmail(t *testing.T) {
	d := testDeps(t)
	ctx := context.Background()

	km := loadCorpusKillmail(t, "103735737")
	removeKillmail(t, d.Pool, km.KillmailID)
	d.ESI = nil

	w := &KillmailWorker{Deps: d}
	if err := w.Work(ctx, testJob(queue.KillmailArgs{
		KillmailID: km.KillmailID, KillmailHash: km.KillmailHash, Killmail: km,
	}, queue.Live)); err != nil {
		t.Fatal(err)
	}

	// Re-delivered with no body and no hash. Reaching the fetch would need the
	// nil ESI client and panic; reaching the cancel would return an error. Only
	// the short-circuit returns nil.
	err := w.Work(ctx, testJob(queue.KillmailArgs{KillmailID: km.KillmailID}, queue.Live))
	if err != nil {
		t.Errorf("a re-delivered killmail returned %v — the existence check did not "+
			"short-circuit, so every repost pays for a parse or a fetch", err)
	}
}
