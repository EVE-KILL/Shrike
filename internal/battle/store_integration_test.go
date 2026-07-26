package battle

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func battleTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"
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

func TestClearWindowRemovesOverlapsAndTheirChildren(t *testing.T) {
	pool := battleTestPool(t)
	ctx := context.Background()
	const systemID int32 = 2_000_000_001
	start := time.Date(2099, 1, 1, 12, 0, 0, 0, time.UTC)

	cleanup := func() {
		rows, _ := pool.Query(ctx,
			`SELECT battle_id FROM battles WHERE solar_system_id = $1`,
			systemID)
		var ids []int64
		if rows != nil {
			for rows.Next() {
				var id int64
				_ = rows.Scan(&id)
				ids = append(ids, id)
			}
			rows.Close()
		}
		if len(ids) > 0 {
			tx, err := pool.Begin(ctx)
			if err == nil {
				_ = deleteBattles(ctx, tx, ids)
				_ = tx.Commit(ctx)
			}
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	autoID, err := Store(ctx, pool, &Detected{
		SolarSystemID:   systemID,
		Start:           start,
		End:             start.Add(time.Hour),
		DurationMinutes: 60,
		KillCount:       10,
		IskDestroyed:    100,
		Teams: [2]Team{
			{Entries: []TeamEntry{{CorporationID: 10_000_001}}},
			{Entries: []TeamEntry{{CorporationID: 10_000_002}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	teamRows, err := pool.Query(ctx,
		`SELECT id FROM battle_teams WHERE battle_id = $1 ORDER BY id`,
		autoID)
	if err != nil {
		t.Fatal(err)
	}
	var teamIDs []int64
	for teamRows.Next() {
		var id int64
		if err := teamRows.Scan(&id); err != nil {
			teamRows.Close()
			t.Fatal(err)
		}
		teamIDs = append(teamIDs, id)
	}
	teamRows.Close()
	if len(teamIDs) != 2 {
		t.Fatalf("stored battle has %d teams, want 2", len(teamIDs))
	}

	var customID int64
	if err := pool.QueryRow(ctx, `
        INSERT INTO battles (
            solar_system_id, start_time, end_time, is_custom)
        VALUES ($1,$2,$3,true)
        RETURNING battle_id`,
		systemID,
		start.Add(10*time.Minute),
		start.Add(40*time.Minute),
	).Scan(&customID); err != nil {
		t.Fatal(err)
	}

	// The auto battle began before the re-scan but overlaps it. A start-only
	// predicate misses this exact case and leaves a duplicate after detection.
	cleared, err := ClearWindow(
		ctx,
		pool,
		start.Add(30*time.Minute),
		start.Add(90*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(cleared) != 1 || cleared[0] != autoID {
		t.Fatalf("cleared ids = %v, want only auto battle %d", cleared, autoID)
	}

	var auto, custom, teams, members int
	if err := pool.QueryRow(ctx, `
        SELECT
            count(*) FILTER (WHERE battle_id = $1),
            count(*) FILTER (WHERE battle_id = $2)
        FROM battles`,
		autoID,
		customID,
	).Scan(&auto, &custom); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
        SELECT
            (SELECT count(*) FROM battle_teams WHERE battle_id = $1),
            (SELECT count(*) FROM battle_team_members
             WHERE battle_team_id = ANY($2::bigint[]))`,
		autoID,
		teamIDs,
	).Scan(&teams, &members); err != nil {
		t.Fatal(err)
	}
	if auto != 0 || teams != 0 || members != 0 {
		t.Errorf("auto battle remnants: battle=%d teams=%d members=%d", auto, teams, members)
	}
	if custom != 1 {
		t.Errorf("custom battle count = %d, want it preserved", custom)
	}
}

func TestLoadSystemExcludesNPCsAndAppliesCustomPrices(t *testing.T) {
	pool := battleTestPool(t)
	ctx := context.Background()

	const (
		systemID int32 = 2_000_000_002
		typeID   int32 = 2_000_000_003
		playerID int64 = 2_000_000_101
		npcID    int64 = 2_000_000_102
	)
	at := time.Date(2099, 2, 1, 12, 0, 0, 0, time.UTC)

	cleanup := func() {
		_, _ = pool.Exec(ctx,
			`DELETE FROM killmail_attackers WHERE killmail_id = ANY($1::bigint[])`,
			[]int64{playerID, npcID})
		_, _ = pool.Exec(ctx,
			`DELETE FROM killmails WHERE killmail_id = ANY($1::bigint[])`,
			[]int64{playerID, npcID})
		_, _ = pool.Exec(ctx,
			`DELETE FROM prices WHERE type_id = $1 AND region_id = 10000002`,
			typeID)
		_, _ = pool.Exec(ctx,
			`DELETE FROM custom_prices WHERE type_id = $1`,
			typeID)
	}
	cleanup()
	t.Cleanup(cleanup)

	if _, err := pool.Exec(ctx, `
        INSERT INTO custom_prices (type_id, date, price)
        VALUES ($1, '9999-12-31', 1000)`,
		typeID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
        INSERT INTO prices (
            type_id, region_id, date, average,
            highest, lowest, order_count, volume)
        VALUES ($1, 10000002, '2099-02-01', 200, 200, 200, 1, 1)`,
		typeID,
	); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		id    int64
		isNPC bool
		value float64
	}{
		{playerID, false, 500},
		{npcID, true, 900},
	} {
		if _, err := pool.Exec(ctx, `
            INSERT INTO killmails (
                killmail_id, killmail_time, killmail_hash,
                solar_system_id, victim_ship_type_id,
                total_value, is_npc)
            VALUES ($1,$2,'battle-load-test',$3,$4,$5,$6)`,
			fixture.id,
			at,
			systemID,
			typeID,
			fixture.value,
			fixture.isNPC,
		); err != nil {
			t.Fatal(err)
		}
	}

	kills, _, err := LoadSystem(
		ctx,
		pool,
		systemID,
		at.Add(-time.Hour),
		at.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 1 || kills[0].KillmailID != playerID {
		t.Fatalf("loaded killmails = %+v, want only player kill %d", kills, playerID)
	}
	if kills[0].TotalValue != 1300 {
		t.Errorf("adjusted value = %.0f, want 500 + (1000 - 200) = 1300",
			kills[0].TotalValue)
	}
}
