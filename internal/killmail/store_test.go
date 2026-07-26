package killmail

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Storage tests, against a real Postgres.
//
// The corpus is written, read back, and diffed column by column. That is the
// only way to catch the errors this layer actually makes: a column left out of
// an INSERT, a zero stored where NULL was meant, a parent index that survives
// the round trip as something else.
//
// The corpus killmails are real, and the local database may well already hold
// them, so every id is shifted into a reserved band. Nothing the test writes can
// collide with imported data, and it cleans up after itself either way.
//
// Skipped when Postgres is unreachable.

const storeTestDSN = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"

// testIDOffset moves a corpus killmail into a band no real kill occupies. The
// column is a 32-bit integer and live ids are around 137 million, so this leaves
// room at both ends.
const testIDOffset int64 = 1_900_000_000

func storePool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = storeTestDSN
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

// shift moves a parsed killmail into the reserved band, ids on its children
// included.
func shift(p *Parsed) *Parsed {
	out := *p
	out.Killmail.KillmailID += testIDOffset

	out.Attackers = make([]Attacker, len(p.Attackers))
	copy(out.Attackers, p.Attackers)
	for i := range out.Attackers {
		out.Attackers[i].KillmailID += testIDOffset
	}

	out.Items = make([]Item, len(p.Items))
	copy(out.Items, p.Items)
	for i := range out.Items {
		out.Items[i].KillmailID += testIDOffset
	}
	return &out
}

func purge(t *testing.T, pool *pgxpool.Pool, ids []int64) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{"killmail_items", "killmail_attackers", "killmail_processing", "killmails"} {
		if _, err := pool.Exec(ctx,
			`DELETE FROM `+table+` WHERE killmail_id = ANY($1::int[])`, ids); err != nil {
			t.Fatalf("purge %s: %v", table, err)
		}
	}
}

// parseCorpus parses the whole corpus into the reserved id band.
func parseCorpus(t *testing.T) []*Parsed {
	t.Helper()
	cache, f := loadFixture(t)
	ctx := context.Background()

	var out []*Parsed
	for _, entry := range loadCorpus(t) {
		date := entry.esi.KillmailTime.UTC().Format("2006-01-02")
		parsed, err := Parse(ctx, cache, f.pricesFor(t, cache, date), &entry.esi, entry.esi.KillmailHash, 0)
		if err != nil {
			t.Fatalf("%d: %v", entry.id, err)
		}
		out = append(out, shift(parsed))
	}
	return out
}

func idsOf(batch []*Parsed) []int64 {
	out := make([]int64, len(batch))
	for i, p := range batch {
		out[i] = p.Killmail.KillmailID
	}
	return out
}

// The whole corpus through COPY and back, diffed column by column. Anything the
// insert drops or the load misreads shows up here.
func TestCorpusRoundTripsThroughPostgres(t *testing.T) {
	pool := storePool(t)
	batch := parseCorpus(t)
	ids := idsOf(batch)

	purge(t, pool, ids)
	t.Cleanup(func() { purge(t, pool, ids) })

	ctx := context.Background()
	res, err := InsertBatch(ctx, pool, batch)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if res.Killmails != int64(len(batch)) {
		t.Errorf("inserted %d killmails, want %d", res.Killmails, len(batch))
	}

	var wantAttackers, wantItems int64
	for _, p := range batch {
		wantAttackers += int64(len(p.Attackers))
		wantItems += int64(len(p.Items))
	}
	if res.Attackers != wantAttackers {
		t.Errorf("inserted %d attackers, want %d", res.Attackers, wantAttackers)
	}
	if res.Items != wantItems {
		t.Errorf("inserted %d items, want %d", res.Items, wantItems)
	}

	for _, want := range batch {
		got, err := Load(ctx, pool, want.Killmail.KillmailID)
		if err != nil {
			t.Errorf("load %d: %v", want.Killmail.KillmailID, err)
			continue
		}
		// security_status is a `real` column, so it loses precision on the way
		// in; a relative tolerance covers that without hiding a wrong value.
		diffs := Diff(want, got, 1e-6)
		if len(diffs) != 0 {
			t.Errorf("killmail %d did not round-trip: %v",
				want.Killmail.KillmailID, diffs[:min(4, len(diffs))])
		}
	}

	// The processing ledger is what the derived-effect machinery reads; a
	// killmail without one is invisible to it.
	var ledger int64
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM killmail_processing WHERE killmail_id = ANY($1::int[])`, ids).Scan(&ledger); err != nil {
		t.Fatal(err)
	}
	if ledger != int64(len(batch)) {
		t.Errorf("%d processing rows for %d killmails", ledger, len(batch))
	}
}

// The archives overlap heavily with what the live queue already stored, so
// re-importing has to be free rather than an error or a duplicate.
func TestInsertBatchIsIdempotent(t *testing.T) {
	pool := storePool(t)
	batch := parseCorpus(t)[:20]
	ids := idsOf(batch)

	purge(t, pool, ids)
	t.Cleanup(func() { purge(t, pool, ids) })

	ctx := context.Background()
	if _, err := InsertBatch(ctx, pool, batch); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if _, err := InsertBatch(ctx, pool, batch); err != nil {
		t.Fatalf("second insert: %v", err)
	}

	var killmails, attackers, items int64
	if err := pool.QueryRow(ctx, `
        SELECT (SELECT count(*) FROM killmails WHERE killmail_id = ANY($1::int[])),
               (SELECT count(*) FROM killmail_attackers WHERE killmail_id = ANY($1::int[])),
               (SELECT count(*) FROM killmail_items WHERE killmail_id = ANY($1::int[]))`,
		ids).Scan(&killmails, &attackers, &items); err != nil {
		t.Fatal(err)
	}

	if killmails != int64(len(batch)) {
		t.Errorf("%d killmail rows after two inserts of %d", killmails, len(batch))
	}
	var wantAttackers, wantItems int64
	for _, p := range batch {
		wantAttackers += int64(len(p.Attackers))
		wantItems += int64(len(p.Items))
	}
	if attackers != wantAttackers || items != wantItems {
		t.Errorf("children duplicated: %d attackers (want %d), %d items (want %d)",
			attackers, wantAttackers, items, wantItems)
	}
}

// Two sources routinely deliver the same kill seconds apart. The loser of that
// race must be told, not left to assume it wrote.
func TestInsertReportsAnAlreadyStoredKillmail(t *testing.T) {
	pool := storePool(t)
	batch := parseCorpus(t)[:1]
	ids := idsOf(batch)

	purge(t, pool, ids)
	t.Cleanup(func() { purge(t, pool, ids) })

	ctx := context.Background()
	stored, err := Insert(ctx, pool, batch[0])
	if err != nil {
		t.Fatal(err)
	}
	if !stored {
		t.Fatal("the first insert reported the killmail as already present")
	}

	stored, err = Insert(ctx, pool, batch[0])
	if err != nil {
		t.Fatalf("a duplicate insert errored instead of reporting: %v", err)
	}
	if stored {
		t.Error("the second insert claimed to have stored the killmail again")
	}
}

func TestDeleteRemovesEveryChild(t *testing.T) {
	pool := storePool(t)
	// A mail with plenty of both children.
	var batch []*Parsed
	for _, p := range parseCorpus(t) {
		if len(p.Items) > 10 && len(p.Attackers) > 1 {
			batch = append(batch, p)
			break
		}
	}
	if len(batch) == 0 {
		t.Skip("no suitable killmail in the corpus")
	}
	ids := idsOf(batch)

	purge(t, pool, ids)
	t.Cleanup(func() { purge(t, pool, ids) })

	ctx := context.Background()
	if _, err := InsertBatch(ctx, pool, batch); err != nil {
		t.Fatal(err)
	}
	if err := Delete(ctx, pool, ids[0]); err != nil {
		t.Fatalf("delete: %v", err)
	}

	for _, table := range []string{"killmails", "killmail_attackers", "killmail_items", "killmail_processing"} {
		var n int64
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM `+table+` WHERE killmail_id = $1`, ids[0]).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%d rows left in %s after delete", n, table)
		}
	}

	if _, err := Load(ctx, pool, ids[0]); !errors.Is(err, ErrNotStored) {
		t.Errorf("loading a deleted killmail returned %v, want ErrNotStored", err)
	}
}

// A war kill the live queue got to first has a null war_id, because the public
// killmail endpoint never states one. Filling it in is the only way a war ever
// shows activity — and it must not overwrite an attribution already made.
func TestAssignWarsFillsOnlyNulls(t *testing.T) {
	pool := storePool(t)
	batch := parseCorpus(t)[:3]
	ids := idsOf(batch)

	purge(t, pool, ids)
	t.Cleanup(func() { purge(t, pool, ids) })

	ctx := context.Background()
	if _, err := InsertBatch(ctx, pool, batch); err != nil {
		t.Fatal(err)
	}

	// One already belongs to a war; the archive claims a different one for it.
	const existingWar, archiveWar = 111111, 222222
	if _, err := pool.Exec(ctx,
		`UPDATE killmails SET war_id = $2 WHERE killmail_id = $1`, ids[0], existingWar); err != nil {
		t.Fatal(err)
	}

	for _, p := range batch {
		p.Killmail.WarID = archiveWar
	}
	assigned, err := AssignWars(ctx, pool, batch)
	if err != nil {
		t.Fatal(err)
	}
	if assigned != 2 {
		t.Errorf("assigned %d war ids, want 2 (the third was already attributed)", assigned)
	}

	var got int32
	if err := pool.QueryRow(ctx,
		`SELECT war_id FROM killmails WHERE killmail_id = $1`, ids[0]).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != existingWar {
		t.Errorf("an existing war attribution was overwritten: %d", got)
	}

	for _, id := range ids[1:] {
		if err := pool.QueryRow(ctx,
			`SELECT war_id FROM killmails WHERE killmail_id = $1`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != archiveWar {
			t.Errorf("killmail %d has war_id %d, want %d", id, got, archiveWar)
		}
	}
}

// Zero-means-absent is a convention of the Go structs, not of the database.
// Storing a literal zero would make "no alliance" indistinguishable from
// alliance 0 in every query that joins on it.
func TestAbsentIDsAreStoredAsNull(t *testing.T) {
	pool := storePool(t)

	var batch []*Parsed
	for _, p := range parseCorpus(t) {
		if p.Killmail.VictimAllianceID == 0 && p.Killmail.VictimShipTypeID != 0 {
			batch = append(batch, p)
			break
		}
	}
	if len(batch) == 0 {
		t.Skip("every corpus victim has an alliance")
	}
	ids := idsOf(batch)

	purge(t, pool, ids)
	t.Cleanup(func() { purge(t, pool, ids) })

	ctx := context.Background()
	if _, err := InsertBatch(ctx, pool, batch); err != nil {
		t.Fatal(err)
	}

	var allianceID *int32
	var warID *int32
	if err := pool.QueryRow(ctx,
		`SELECT victim_alliance_id, war_id FROM killmails WHERE killmail_id = $1`, ids[0]).
		Scan(&allianceID, &warID); err != nil {
		t.Fatal(err)
	}
	if allianceID != nil {
		t.Errorf("an absent alliance was stored as %d", *allianceID)
	}
	if warID != nil {
		t.Errorf("an absent war was stored as %d", *warID)
	}

	// And a top-level item's parent must be NULL, not zero — item 0 is a real
	// item and is frequently a parent.
	var nullParents, zeroParents int64
	if err := pool.QueryRow(ctx, `
        SELECT count(*) FILTER (WHERE parent_index IS NULL),
               count(*) FILTER (WHERE parent_index = 0)
        FROM killmail_items WHERE killmail_id = $1`, ids[0]).Scan(&nullParents, &zeroParents); err != nil {
		t.Fatal(err)
	}
	if len(batch[0].Items) > 0 && nullParents == 0 {
		t.Error("no top-level items stored a NULL parent_index")
	}
}
