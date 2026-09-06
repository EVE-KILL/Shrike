package intelrollup

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRefreshDelayUsesUTCDay(t *testing.T) {
	now := time.Date(2026, 9, 6, 0, 1, 0, 0, time.UTC)
	for _, tc := range []struct {
		day  string
		want time.Duration
	}{
		{"2026-09-06T00:00:00Z", 5 * time.Minute},
		{"2026-09-05T00:00:00Z", 5 * time.Minute},
		{"2026-09-04T23:59:59Z", 6 * time.Hour},
		{"2026-06-12T12:00:00Z", 6 * time.Hour},
		{"2026-09-05T01:00:00+02:00", 6 * time.Hour},
	} {
		day, err := time.Parse(time.RFC3339, tc.day)
		if err != nil {
			t.Fatal(err)
		}
		if got := RefreshDelay(day, now); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.day, got, tc.want)
		}
	}
}

func TestCharacterDailyBoundsStayInline(t *testing.T) {
	if !strings.Contains(characterDailySQL, "bounds AS NOT MATERIALIZED") {
		t.Fatal("date bounds must stay visible to the planner")
	}
}

func TestInlineBoundsPreserveDailyFacts(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires explicit disposable TEST_DATABASE_URL")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	// Temporary shadows keep this test independent of other fixture data.
	for _, table := range []string{"killmails", "killmail_attackers", "killmail_items", "solar_systems", "character_intel_daily"} {
		if _, err := tx.Exec(ctx, "CREATE TEMP TABLE "+table+" (LIKE public."+table+" INCLUDING DEFAULTS) ON COMMIT DROP"); err != nil {
			t.Fatal(err)
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO killmails(killmail_id,killmail_time,killmail_hash,solar_system_id,victim_character_id,total_value,attacker_count,victim_alliance_id)
		VALUES (1,'2026-09-05 23:59Z','test',1,101,1000,2,99),
		       (2,'2026-09-06 00:01Z','test',1,102,1000,2,99);
		INSERT INTO killmail_attackers(killmail_id,attacker_index,character_id,killmail_time,ship_type_id)
		VALUES (1,0,201,'2026-09-05 23:59Z',45534), (2,0,202,'2026-09-06 00:01Z',45534);
		INSERT INTO killmail_items(killmail_id,item_index,type_id,flag_id)
		VALUES (1,0,21096,11);`)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	readFacts := func(query string) string {
		t.Helper()
		if _, err := tx.Exec(ctx, "DELETE FROM character_intel_daily"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, query, day); err != nil {
			t.Fatal(err)
		}
		var facts string
		if err := tx.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(d) ORDER BY character_id)::text FROM character_intel_daily d`).Scan(&facts); err != nil {
			t.Fatal(err)
		}
		return facts
	}
	old := readFacts(strings.Replace(characterDailySQL, "bounds AS NOT MATERIALIZED", "bounds AS MATERIALIZED", 1))
	got := readFacts(characterDailySQL)
	if old != got {
		t.Fatalf("facts changed: old=%s new=%s", old, got)
	}
	var losses, cyno, bait int
	if err := tx.QueryRow(ctx, "SELECT losses,cyno_losses,baited_deaths FROM character_intel_daily WHERE character_id=101").Scan(&losses, &cyno, &bait); err != nil {
		t.Fatal(err)
	}
	if losses != 1 || cyno != 1 || bait != 1 {
		t.Fatalf("lost boundary/cyno facts: %d %d %d", losses, cyno, bait)
	}
	var count int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM character_intel_daily").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("included next-day characters: %d", count)
	}
}
