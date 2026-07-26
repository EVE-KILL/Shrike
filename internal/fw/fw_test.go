package fw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// The whole point of this package is one property: applying the same snapshot
// twice must write nothing the second time. Production's TypeScript fails it —
// fw_systems has not advanced since 2026-04-11 while fw_system_history has
// grown to 88,376 rows describing 53 real transitions — so the tests here are
// less about proving the Go works than about making sure it never regresses
// into the same shape.

const fwTestDSN = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"

// Reserved system ids above anything CCP has issued, so nothing these tests
// write can collide with imported data on a shared development database.
const testSystemBase = 39_100_000

// A Redis database of this package's own. `go test ./...` runs packages in
// parallel and the ESI client's buckets live in a shared keyspace; sharing one
// index between packages produced a phantom failure once already.
const testRedisDB = 12

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = fwTestDSN
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

// testSystems reserves a band of system ids and clears them before and after.
func testSystems(t *testing.T, pool *pgxpool.Pool, n int32) []int32 {
	t.Helper()

	ids := make([]int32, n)
	for i := range ids {
		ids[i] = testSystemBase + int32(i)
	}

	clear := func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM fw_system_history WHERE solar_system_id = ANY($1::int[])`, ids)
		_, _ = pool.Exec(ctx, `DELETE FROM fw_systems WHERE solar_system_id = ANY($1::int[])`, ids)
	}
	clear()
	t.Cleanup(clear)
	return ids
}

// fwServer serves a fixed faction warfare system list through a real ESI
// pipeline — real bucket, real singleflight — so the importer is exercised the
// way it runs rather than against a stub client.
func fwServer(t *testing.T, systems []esi.FwSystem) *esi.Client {
	t.Helper()

	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: addr, DB: testRedisDB, MaxRetries: -1, DialTimeout: 2 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		t.Skipf("no test redis at %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(systems)
	}))
	t.Cleanup(srv.Close)

	client := esi.NewForTest(srv.URL, "shrike-test/1.0", rdb, rdb)
	t.Cleanup(client.Close)
	return client
}

func counts(t *testing.T, pool *pgxpool.Pool, ids []int32) (systems, history int64) {
	t.Helper()
	err := pool.QueryRow(context.Background(),
		`SELECT (SELECT count(*) FROM fw_systems WHERE solar_system_id = ANY($1::int[])),
		        (SELECT count(*) FROM fw_system_history WHERE solar_system_id = ANY($1::int[]))`,
		ids).Scan(&systems, &history)
	if err != nil {
		t.Fatal(err)
	}
	return
}

// The regression test for the production bug. A second identical snapshot must
// write nothing at all.
func TestIdenticalSnapshotIsANoOp(t *testing.T) {
	pool := testPool(t)
	ids := testSystems(t, pool, 3)
	ctx := context.Background()

	snapshot := []esi.FwSystem{
		{SolarSystemID: ids[0], OwnerFactionID: 500001, OccupierFactionID: 500001,
			Contested: "uncontested", VictoryPoints: 0, VictoryPointsThreshold: 3000},
		{SolarSystemID: ids[1], OwnerFactionID: 500002, OccupierFactionID: 500004,
			Contested: "contested", VictoryPoints: 1500, VictoryPointsThreshold: 3000},
		{SolarSystemID: ids[2], OwnerFactionID: 500003, OccupierFactionID: 500003,
			Contested: "uncontested", VictoryPoints: 0, VictoryPointsThreshold: 3000},
	}
	client := fwServer(t, snapshot)

	first, err := ImportSystems(ctx, pool, client)
	if err != nil {
		t.Fatal(err)
	}
	if first.Rows != 3 {
		t.Errorf("first import wrote %d rows, want 3", first.Rows)
	}
	// Nothing was known before, so nothing can have flipped.
	if first.Flips != 0 {
		t.Errorf("first import recorded %d flips, want 0 — a system seen for the "+
			"first time has nothing to have changed from", first.Flips)
	}

	second, err := ImportSystems(ctx, pool, client)
	if err != nil {
		t.Fatal(err)
	}
	if second.Rows != 0 {
		t.Errorf("an unchanged snapshot wrote %d rows — this is the production bug: "+
			"the current-state table is not advancing", second.Rows)
	}
	if second.Flips != 0 {
		t.Errorf("an unchanged snapshot recorded %d flips — every run would re-detect "+
			"the same transitions forever, which is how fw_system_history reached "+
			"88,376 rows for 53 real events", second.Flips)
	}

	systems, history := counts(t, pool, ids)
	if systems != 3 {
		t.Errorf("%d current rows, want 3", systems)
	}
	if history != 0 {
		t.Errorf("%d history rows after two identical snapshots, want 0", history)
	}
}

// The current-state table has to actually change — the specific thing the
// TypeScript fails to do.
func TestOccupierChangeAdvancesCurrentState(t *testing.T) {
	pool := testPool(t)
	ids := testSystems(t, pool, 1)
	ctx := context.Background()

	before := []esi.FwSystem{{
		SolarSystemID: ids[0], OwnerFactionID: 500001, OccupierFactionID: 500001,
		Contested: "uncontested", VictoryPointsThreshold: 3000,
	}}
	if _, err := ImportSystems(ctx, pool, fwServer(t, before)); err != nil {
		t.Fatal(err)
	}

	after := []esi.FwSystem{{
		SolarSystemID: ids[0], OwnerFactionID: 500001, OccupierFactionID: 500004,
		Contested: "uncontested", VictoryPointsThreshold: 3000,
	}}
	res, err := ImportSystems(ctx, pool, fwServer(t, after))
	if err != nil {
		t.Fatal(err)
	}
	if res.Rows != 1 {
		t.Errorf("a change of occupier wrote %d rows", res.Rows)
	}
	if res.Flips != 1 {
		t.Errorf("a change of occupier recorded %d flips, want 1", res.Flips)
	}

	var occupier int32
	var updatedAt time.Time
	if err := pool.QueryRow(ctx,
		`SELECT occupier_faction_id, updated_at FROM fw_systems WHERE solar_system_id = $1`,
		ids[0]).Scan(&occupier, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if occupier != 500004 {
		t.Fatalf("stored occupier = %d, want 500004 — the current-state table is not "+
			"advancing, which is the production bug this importer exists to avoid", occupier)
	}
	if time.Since(updatedAt) > time.Minute {
		t.Errorf("updated_at was not moved: %v", updatedAt)
	}

	// And exactly one history row, recording the transition that happened.
	var oldID, newID int32
	if err := pool.QueryRow(ctx, `
        SELECT old_occupier_faction_id, new_occupier_faction_id
        FROM fw_system_history WHERE solar_system_id = $1`, ids[0]).Scan(&oldID, &newID); err != nil {
		t.Fatal(err)
	}
	if oldID != 500001 || newID != 500004 {
		t.Errorf("history recorded %d → %d, want 500001 → 500004", oldID, newID)
	}
}

// Victory points move constantly without the system changing hands. That must
// update the row but must not be recorded as a flip.
func TestVictoryPointsChangeIsNotAFlip(t *testing.T) {
	pool := testPool(t)
	ids := testSystems(t, pool, 1)
	ctx := context.Background()

	before := []esi.FwSystem{{
		SolarSystemID: ids[0], OwnerFactionID: 500001, OccupierFactionID: 500001,
		Contested: "contested", VictoryPoints: 100, VictoryPointsThreshold: 3000,
	}}
	if _, err := ImportSystems(ctx, pool, fwServer(t, before)); err != nil {
		t.Fatal(err)
	}

	after := []esi.FwSystem{{
		SolarSystemID: ids[0], OwnerFactionID: 500001, OccupierFactionID: 500001,
		Contested: "contested", VictoryPoints: 2900, VictoryPointsThreshold: 3000,
	}}
	res, err := ImportSystems(ctx, pool, fwServer(t, after))
	if err != nil {
		t.Fatal(err)
	}
	if res.Rows != 1 {
		t.Errorf("a victory point change wrote %d rows, want 1", res.Rows)
	}
	if res.Flips != 0 {
		t.Errorf("a victory point change recorded %d flips — the system did not "+
			"change hands, and recording it would fill the history with contest "+
			"progress rather than actual captures", res.Flips)
	}

	_, history := counts(t, pool, ids)
	if history != 0 {
		t.Errorf("%d history rows for a system that never changed hands", history)
	}
}

// A system with no id is not a system.
func TestSystemsWithNoIDAreIgnored(t *testing.T) {
	pool := testPool(t)
	ids := testSystems(t, pool, 1)
	ctx := context.Background()

	client := fwServer(t, []esi.FwSystem{
		{SolarSystemID: ids[0], OwnerFactionID: 500001, OccupierFactionID: 500001},
		{SolarSystemID: 0, OwnerFactionID: 500002, OccupierFactionID: 500002},
	})

	res, err := ImportSystems(ctx, pool, client)
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 2 {
		t.Errorf("seen = %d, want the 2 entries offered", res.Seen)
	}
	if res.Rows != 1 {
		t.Errorf("wrote %d rows, want 1 — a system_id of 0 must be ignored", res.Rows)
	}
}

// PurgeHistoryDuplicates has to remove consecutive repeats and keep the first
// of each run, or it would either leave the noise behind or delete real events.
func TestPurgeHistoryDuplicatesKeepsRealTransitions(t *testing.T) {
	pool := testPool(t)
	ids := testSystems(t, pool, 1)
	ctx := context.Background()

	// A → B recorded five times (the artefact), then B → C once (real), then
	// B → C twice more (artefact again).
	rows := []struct{ old, new int32 }{
		{500001, 500004}, {500001, 500004}, {500001, 500004}, {500001, 500004}, {500001, 500004},
		{500004, 500002},
		{500004, 500002}, {500004, 500002},
	}
	base := time.Now().UTC().Add(-time.Hour)
	for i, r := range rows {
		if _, err := pool.Exec(ctx, `
            INSERT INTO fw_system_history (solar_system_id, old_occupier_faction_id, new_occupier_faction_id, flipped_at)
            VALUES ($1,$2,$3,$4)`,
			ids[0], r.old, r.new, base.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := PurgeHistoryDuplicates(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if removed < 6 {
		t.Errorf("removed %d rows, want at least the 6 consecutive repeats", removed)
	}

	var remaining int64
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM fw_system_history WHERE solar_system_id = $1`, ids[0]).
		Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 2 {
		t.Errorf("%d rows left, want 2 — one per genuine transition", remaining)
	}

	// And specifically the two real ones, in order.
	var got []string
	r, err := pool.Query(ctx, `
        SELECT old_occupier_faction_id || '→' || new_occupier_faction_id
        FROM fw_system_history WHERE solar_system_id = $1 ORDER BY flipped_at`, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for r.Next() {
		var s string
		if err := r.Scan(&s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	if len(got) != 2 || got[0] != "500001→500004" || got[1] != "500004→500002" {
		t.Errorf("surviving transitions = %v, want the two real ones", got)
	}
}
