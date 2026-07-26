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
	if len(got) != 5 {
		t.Fatalf("combat definitions = %d, want 5", len(got))
	}

	got = Filter("veteran_killer", "COMBAT")
	if len(got) != 1 || got[0].ID != "veteran_killer" {
		t.Fatalf("filtered definitions = %#v", got)
	}

	if got := Filter("veteran_killer", "Locations"); len(got) != 0 {
		t.Fatalf("mismatched filters returned %d definitions", len(got))
	}
}

func TestEveryDefinitionHasARebuildQuery(t *testing.T) {
	for _, def := range All {
		query, _, err := countQuery(def)
		if err != nil {
			t.Errorf("%s: %v", def.ID, err)
		}
		if query == "" {
			t.Errorf("%s has an empty rebuild query", def.ID)
		}
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
	defer pool.Close()

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
