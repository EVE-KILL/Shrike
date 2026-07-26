package campaign

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func campaignTestPool(t *testing.T) *pgxpool.Pool {
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

// The integration shape catches the campaign bugs that a query-level unit
// test cannot: equal-valued kills must both contribute ISK, friendly fire is a
// loss but not a kill, and a participant campaign's location still narrows the
// candidate set.
func TestProcessCampaignAggregatesCompleteScopedSet(t *testing.T) {
	pool := campaignTestPool(t)
	ctx := context.Background()

	const campaignID = "gotestcmp00001"
	killmailIDs := []int64{2_000_000_001, 2_000_000_002, 2_000_000_003, 2_000_000_004}
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM campaign_scratch_candidates WHERE campaign_id = $1`, campaignID)
		_, _ = pool.Exec(ctx, `DELETE FROM campaign_scratch_killmails WHERE campaign_id = $1`, campaignID)
		_, _ = pool.Exec(ctx, `DELETE FROM campaigns WHERE campaign_id = $1`, campaignID)
		_, _ = pool.Exec(ctx, `DELETE FROM killmail_attackers WHERE killmail_id = ANY($1::bigint[])`, killmailIDs)
		_, _ = pool.Exec(ctx, `DELETE FROM killmails WHERE killmail_id = ANY($1::bigint[])`, killmailIDs)
	}
	cleanup()
	t.Cleanup(cleanup)

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `
        INSERT INTO campaigns (
            campaign_id, name, created_by_character_id,
            start_time, end_time, location)
        VALUES ($1, 'Go campaign aggregate test', 90000001,
                $2, $3, '{"systemIds":[30000142]}'::jsonb)`,
		campaignID,
		now.Add(-time.Hour),
		now.Add(time.Hour),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
        INSERT INTO campaign_sides (campaign_id, side_index, name)
        VALUES ($1, 0, 'Attackers'), ($1, 1, 'Defenders')`,
		campaignID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
        INSERT INTO campaign_side_entities (
            campaign_id, side_index, entity_type, entity_id)
        VALUES ($1, 0, $2, 10000100), ($1, 1, $2, 10000200)`,
		campaignID,
		EntityCorporation,
	); err != nil {
		t.Fatal(err)
	}

	// Two real kills of exactly the same value, one same-side loss, and one
	// otherwise-matching kill outside the campaign's system.
	type fixture struct {
		id           int64
		system       int32
		victimCorp   int32
		attackerCorp int32
		value        float64
		character    int32
	}
	fixtures := []fixture{
		{killmailIDs[0], 30000142, 10000200, 10000100, 100, 91000001},
		{killmailIDs[1], 30000142, 10000200, 10000100, 100, 91000002},
		{killmailIDs[2], 30000142, 10000100, 10000100, 100, 91000003},
		{killmailIDs[3], 30000144, 10000200, 10000100, 500, 91000004},
	}
	for i, fixture := range fixtures {
		at := now.Add(time.Duration(i-10) * time.Minute)
		if _, err := pool.Exec(ctx, `
            INSERT INTO killmails (
                killmail_id, killmail_time, killmail_hash,
                solar_system_id, constellation_id, region_id,
                victim_character_id, victim_corporation_id,
                total_value, attacker_count)
            VALUES ($1,$2,$3,$4,20000020,10000002,$5,$6,$7,1)`,
			fixture.id,
			at,
			"campaign-test",
			fixture.system,
			int32(92000000+i),
			fixture.victimCorp,
			fixture.value,
		); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
            INSERT INTO killmail_attackers (
                killmail_id, attacker_index, character_id,
                corporation_id, damage_done, final_blow, killmail_time)
            VALUES ($1,0,$3,$4,100,true,$2)`,
			fixture.id,
			at,
			fixture.character,
			fixture.attackerCorp,
		); err != nil {
			t.Fatal(err)
		}
	}

	result, err := Process(ctx, pool, campaignID)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Killmails != 3 {
		t.Fatalf("result = %#v, want 3 in-system killmails", result)
	}

	type totals struct {
		side            int16
		kills, losses   int64
		destroyed, lost float64
	}
	rows, err := pool.Query(ctx, `
        SELECT side_index, kills, losses, isk_destroyed, isk_lost
        FROM campaign_sides
        WHERE campaign_id = $1
        ORDER BY side_index`,
		campaignID,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []totals
	for rows.Next() {
		var item totals
		if err := rows.Scan(
			&item.side,
			&item.kills,
			&item.losses,
			&item.destroyed,
			&item.lost,
		); err != nil {
			t.Fatal(err)
		}
		got = append(got, item)
	}
	if len(got) != 2 {
		t.Fatalf("side rows = %v, want 2", got)
	}
	if got[0].kills != 2 || got[0].losses != 1 ||
		got[0].destroyed != 200 || got[0].lost != 100 {
		t.Errorf("side 0 = %+v, want 2/1 kills/losses and 200/100 ISK", got[0])
	}
	if got[1].kills != 0 || got[1].losses != 2 ||
		got[1].destroyed != 0 || got[1].lost != 200 {
		t.Errorf("side 1 = %+v, want 0/2 kills/losses and 0/200 ISK", got[1])
	}

	var raw []byte
	if err := pool.QueryRow(ctx,
		`SELECT stats FROM campaigns WHERE campaign_id = $1`,
		campaignID,
	).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var stats struct {
		Totals struct {
			KillCount    int64   `json:"killCount"`
			IskDestroyed float64 `json:"iskDestroyed"`
		} `json:"totals"`
	}
	if err := json.Unmarshal(raw, &stats); err != nil {
		t.Fatal(err)
	}
	if stats.Totals.KillCount != 3 || stats.Totals.IskDestroyed != 300 {
		t.Errorf("stats totals = %+v, want 3 killmails worth 300", stats.Totals)
	}
}
