package achievements

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFilterMatchesIDAndCategory(t *testing.T) {
	got := Filter("", "combat")
	if len(got) != 6 {
		t.Fatalf("combat definitions = %d, want 6", len(got))
	}

	got = Filter("veteran_killer", "COMBAT")
	if len(got) != 1 || got[0].ID != "veteran_killer" {
		t.Fatalf("filtered definitions = %#v", got)
	}

	if got := Filter("veteran_killer", "Locations"); len(got) != 0 {
		t.Fatalf("mismatched filters returned %d definitions", len(got))
	}
}

func TestEveryDefinitionHasARebuildSource(t *testing.T) {
	sources, err := rebuildSources(All)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 129 {
		t.Fatalf("shared count sources = %d, want 129", len(sources))
	}

	definitions := 0
	for _, source := range sources {
		if source.Index == 0 || len(source.Definitions) == 0 {
			t.Errorf("invalid rebuild source: %+v", source)
		}
		definitions += len(source.Definitions)
	}
	if definitions != len(All) {
		t.Errorf("sources contain %d definitions, want %d", definitions, len(All))
	}
}

func TestRebuildRemovesCountersThatNoLongerHaveSourceRows(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		t.Skipf("no test database: %v", err)
	}

	const (
		characterID = int32(2_140_000_001)
		achievement = "audit_rebuild_zero"
	)
	ctx := context.Background()
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM entity_achievements WHERE achievement_id = $1`, achievement)
		_, _ = pool.Exec(ctx, `DELETE FROM stats WHERE entity_type = 0 AND entity_id = $1`, characterID)
		_, _ = pool.Exec(ctx, `DELETE FROM characters WHERE character_id = $1`, characterID)
	}
	cleanup()
	t.Cleanup(cleanup)

	if _, err := pool.Exec(ctx, `
		INSERT INTO characters (character_id, name, achievement_points)
		VALUES ($1, 'Achievement rebuild audit', 500)`, characterID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_achievements (
			entity_id, achievement_id, current_count, threshold,
			completion_tiers, is_completed, points
		) VALUES ($1, $2, 50, 10, 5, true, 500)`,
		characterID, achievement); err != nil {
		t.Fatal(err)
	}

	res, err := Rebuild(ctx, pool, []Definition{{
		ID:         achievement,
		Threshold:  10,
		BasePoints: 100,
		Trigger:    TriggerShipKills,
		GroupIDs:   []int32{2_140_000_001},
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Rows != 0 || res.Removed != 1 {
		t.Errorf("rebuild result = %+v, want 0 upserted and 1 removed", res)
	}

	var achievementRows int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM entity_achievements
		WHERE entity_id = $1 AND achievement_id = $2`,
		characterID, achievement).Scan(&achievementRows); err != nil {
		t.Fatal(err)
	}
	if achievementRows != 0 {
		t.Errorf("stale achievement rows = %d, want 0", achievementRows)
	}
}

func TestRebuildSharesAllCountFamilies(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	const (
		attackerID = int32(2_140_000_011)
		victimID   = int32(2_140_000_012)
		killmailID = int32(-2_140_000_011)
		systemID   = int32(-2_140_000_012)
		shipGroup  = int32(-2_140_000_013)
	)
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM stats WHERE entity_type = 0 AND entity_id = $1`, attackerID)
		_, _ = pool.Exec(ctx, `DELETE FROM killmail_attackers WHERE killmail_id = $1`, killmailID)
		_, _ = pool.Exec(ctx, `DELETE FROM killmails WHERE killmail_id = $1`, killmailID)
		_, _ = pool.Exec(ctx, `DELETE FROM solar_systems WHERE solar_system_id = $1`, systemID)
	}
	cleanup()
	t.Cleanup(cleanup)

	if _, err := pool.Exec(ctx,
		`INSERT INTO solar_systems (solar_system_id, system_name, security)
		 VALUES ($1, 'Achievement rebuild audit', 42)`, systemID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO killmails (
			killmail_id, killmail_time, killmail_hash, solar_system_id,
			victim_character_id, victim_ship_group_id,
			total_value, is_npc, is_solo
		) VALUES ($1, '1970-01-01T00:00:00Z', 'achievement-rebuild-audit', $2,
		          $3, $4, 1e300, false, true)`,
		killmailID, systemID, victimID, shipGroup); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO killmail_attackers (
			killmail_id, killmail_time, attacker_index, character_id, final_blow
		) VALUES ($1, '1970-01-01T00:00:00Z', 0, $2, true)`,
		killmailID, attackerID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO stats (
			entity_type, entity_id, period_type, period_start, final_blows, solo_kills
		) VALUES (0, $1, 2, '1970-01-01', 7, 3)`, attackerID); err != nil {
		t.Fatal(err)
	}

	definitions := []Definition{
		{ID: "audit_shared_final", Threshold: 1, BasePoints: 1, Trigger: TriggerFinalBlows},
		{ID: "audit_shared_solo", Threshold: 1, BasePoints: 1, Trigger: TriggerSoloKills},
		{ID: "audit_shared_value", Threshold: 1, BasePoints: 1, Trigger: TriggerKillsByValue, MinValue: 1e299},
		{ID: "audit_shared_security", Threshold: 1, BasePoints: 1, Trigger: TriggerKillsBySecurity, MinSec: 41, MaxSec: 43},
		{ID: "audit_shared_ship_kill", Threshold: 1, BasePoints: 1, Trigger: TriggerShipKills, GroupIDs: []int32{shipGroup}},
		{ID: "audit_shared_ship_loss", Threshold: 1, BasePoints: 1, Trigger: TriggerShipLosses, GroupIDs: []int32{shipGroup}},
	}
	sources, err := rebuildSources(definitions)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := loadRebuildCounts(ctx, tx, sources); err != nil {
		t.Fatal(err)
	}

	expected := map[string]struct {
		entity int32
		count  int32
	}{
		"audit_shared_final":     {attackerID, 7},
		"audit_shared_solo":      {attackerID, 3},
		"audit_shared_value":     {attackerID, 1},
		"audit_shared_security":  {attackerID, 1},
		"audit_shared_ship_kill": {attackerID, 1},
		"audit_shared_ship_loss": {victimID, 1},
	}
	for _, source := range sources {
		achievement := source.Definitions[0].ID
		want := expected[achievement]
		var count int32
		if err := tx.QueryRow(ctx, `
			SELECT cnt
			FROM achievement_rebuild_counts
			WHERE source_index = $1 AND character_id = $2`,
			source.Index, want.entity).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want.count {
			t.Errorf("%s count = %d, want %d", achievement, count, want.count)
		}
	}
}
