package everef

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sovereignty is the only importer with real state: it compares each snapshot
// against what is stored and writes only what moved. Getting that wrong is not
// a crash, it is a history log that fills with noise — which is exactly what
// production does, because the TypeScript cron's upsert assigns every column to
// itself and the current-state table therefore never advances. Every run then
// re-detects the same differences forever.
//
// These tests pin the behaviour that avoids it.

func sovSystems(t *testing.T, pool *pgxpool.Pool, n int32) []int32 {
	t.Helper()
	ids := make([]int32, n)
	for i := range ids {
		ids[i] = testSystemBase + int32(i)
	}

	cleanup := func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM sovereignty_history WHERE system_id = ANY($1::int[])`, ids)
		_, _ = pool.Exec(ctx, `DELETE FROM sovereignty WHERE system_id = ANY($1::int[])`, ids)
	}
	cleanup()
	t.Cleanup(cleanup)
	return ids
}

func sovCounts(t *testing.T, pool *pgxpool.Pool, ids []int32) (current, history int64) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM sovereignty WHERE system_id = ANY($1::int[])),
		        (SELECT count(*) FROM sovereignty_history WHERE system_id = ANY($1::int[]))`,
		ids).Scan(&current, &history); err != nil {
		t.Fatal(err)
	}
	return
}

// Applying the same snapshot twice must write nothing the second time. This is
// the regression test for the production bug: there, the second run rewrites
// every row to itself and appends a duplicate history entry, and does so every
// six hours forever.
func TestSovereigntyIdenticalSnapshotIsANoOp(t *testing.T) {
	pool := testPool(t)
	ids := sovSystems(t, pool, 3)
	ctx := context.Background()

	entries := []sovEntry{
		{SystemID: ids[0], AllianceID: 99000001},
		{SystemID: ids[1], CorporationID: 98000001},
		{SystemID: ids[2]}, // unowned
	}

	state, err := loadSovState(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}

	first, err := state.apply(ctx, entries, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if first.Rows != 3 {
		t.Errorf("first snapshot wrote %d rows, want 3", first.Rows)
	}
	// The unowned system is a row but not an event: there is nothing to log.
	if first.Related != 2 {
		t.Errorf("first snapshot logged %d history entries, want 2", first.Related)
	}

	// A fresh state, as a later run would have — the point is that it reads the
	// stored rows and finds nothing to do.
	reloaded, err := loadSovState(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	second, err := reloaded.apply(ctx, entries, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if second.Rows != 0 {
		t.Errorf("an unchanged snapshot wrote %d rows — the current-state table is not advancing", second.Rows)
	}
	if second.Related != 0 {
		t.Errorf("an unchanged snapshot appended %d history entries", second.Related)
	}

	current, history := sovCounts(t, pool, ids)
	if current != 3 {
		t.Errorf("%d current rows, want 3", current)
	}
	if history != 2 {
		t.Errorf("%d history rows after two identical snapshots, want 2", history)
	}
}

// The current-state table has to actually change, which is the specific thing
// the TypeScript cron fails to do.
func TestSovereigntyOwnershipChangeIsRecorded(t *testing.T) {
	pool := testPool(t)
	ids := sovSystems(t, pool, 1)
	ctx := context.Background()

	state, err := loadSovState(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.apply(ctx, []sovEntry{{SystemID: ids[0], AllianceID: 99000001}}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	// The system changes hands.
	reloaded, err := loadSovState(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	res, err := reloaded.apply(ctx, []sovEntry{{SystemID: ids[0], AllianceID: 99000002}}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if res.Rows != 1 {
		t.Errorf("a change of owner wrote %d rows", res.Rows)
	}
	if res.Related != 1 {
		t.Errorf("a change of owner logged %d history entries, want 1", res.Related)
	}

	var owner *int32
	var updatedAt time.Time
	if err := pool.QueryRow(ctx,
		`SELECT alliance_id, updated_at FROM sovereignty WHERE system_id = $1`, ids[0]).
		Scan(&owner, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if owner == nil || *owner != 99000002 {
		t.Fatalf("current owner = %s, want 99000002 — the current-state table is not advancing, "+
			"which is the production bug this importer exists to avoid", fmtOwner(owner))
	}
	if time.Since(updatedAt) > time.Minute {
		t.Errorf("updated_at was not moved: %v", updatedAt)
	}
}

// Losing sovereignty is an event too, and it has to store NULL rather than a
// zero that would read as "alliance 0 holds it".
func TestSovereigntyLossIsRecordedAsNull(t *testing.T) {
	pool := testPool(t)
	ids := sovSystems(t, pool, 1)
	ctx := context.Background()

	state, _ := loadSovState(ctx, pool)
	if _, err := state.apply(ctx, []sovEntry{{SystemID: ids[0], AllianceID: 99000001}}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	reloaded, _ := loadSovState(ctx, pool)
	res, err := reloaded.apply(ctx, []sovEntry{{SystemID: ids[0]}}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if res.Rows != 1 || res.Related != 1 {
		t.Errorf("losing sovereignty wrote %d rows and %d history entries, want 1 and 1", res.Rows, res.Related)
	}

	var owner *int32
	if err := pool.QueryRow(ctx,
		`SELECT alliance_id FROM sovereignty WHERE system_id = $1`, ids[0]).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	if owner != nil {
		t.Errorf("an unowned system stored alliance_id %d", *owner)
	}
}

// Replaying history is not allowed to rewind the live sovereignty map, and a
// second replay of the same snapshot must not duplicate the history event.
func TestSovereigntyHistoricalReplayIsIdempotentAndDoesNotRewindCurrent(t *testing.T) {
	pool := testPool(t)
	ids := sovSystems(t, pool, 1)
	ctx := context.Background()

	currentAt := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO sovereignty (
			system_id, alliance_id, date_added, updated_at
		) VALUES ($1, $2, $3, $3)`,
		ids[0], int32(99000002), currentAt); err != nil {
		t.Fatal(err)
	}

	replayAt := time.Date(2017, 1, 2, 12, 0, 0, 0, time.UTC)
	applyReplay := func() Result {
		t.Helper()
		state, err := loadSovHistoryState(ctx, pool, replayAt)
		if err != nil {
			t.Fatal(err)
		}
		res, err := state.apply(ctx,
			[]sovEntry{{SystemID: ids[0], AllianceID: 99000001}},
			replayAt)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	first := applyReplay()
	if first.Rows != 0 {
		t.Errorf("historical replay rewrote %d current rows, want 0", first.Rows)
	}
	if first.Related != 1 {
		t.Errorf("first replay added %d history rows, want 1", first.Related)
	}

	second := applyReplay()
	if second.Related != 0 {
		t.Errorf("second replay duplicated %d history rows", second.Related)
	}

	var currentOwner int32
	if err := pool.QueryRow(ctx,
		`SELECT alliance_id FROM sovereignty WHERE system_id = $1`, ids[0]).
		Scan(&currentOwner); err != nil {
		t.Fatal(err)
	}
	if currentOwner != 99000002 {
		t.Errorf("current owner = %d after historical replay, want 99000002", currentOwner)
	}

	_, history := sovCounts(t, pool, ids)
	if history != 1 {
		t.Errorf("history rows after two replays = %d, want 1", history)
	}
}

// The whole path, from the published JSON to the two tables.
func TestImportSovereigntyLatest(t *testing.T) {
	pool := testPool(t)
	ids := sovSystems(t, pool, 2)
	ctx := context.Background()

	body := `[
        {"system_id": ` + itoa(ids[0]) + `, "alliance_id": 99000001},
        {"system_id": ` + itoa(ids[1]) + `, "faction_id": 500001},
        {"system_id": 0, "alliance_id": 99000001}
    ]`
	client := jsonServer(t, map[string]string{
		"/sovereignty-map/sovereignty-map-latest.json": body,
	})

	res, err := ImportSovereigntyLatest(ctx, pool, client)
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 3 {
		t.Errorf("seen = %d, want the 3 entries offered", res.Seen)
	}
	// The entry with no system is not a system; EVE uses 0 as "no entity".
	if res.Rows != 2 {
		t.Errorf("wrote %d rows, want 2 — a system_id of 0 must be ignored", res.Rows)
	}

	var faction *int32
	if err := pool.QueryRow(ctx,
		`SELECT faction_id FROM sovereignty WHERE system_id = $1`, ids[1]).Scan(&faction); err != nil {
		t.Fatal(err)
	}
	if faction == nil || *faction != 500001 {
		t.Errorf("faction sovereignty = %s, want 500001", fmtOwner(faction))
	}
}

func itoa(v int32) string { return strconv.Itoa(int(v)) }

// fmtOwner renders a nullable id, because %v on an *int32 prints the address.
func fmtOwner(v *int32) string {
	if v == nil {
		return "NULL"
	}
	return strconv.Itoa(int(*v))
}
