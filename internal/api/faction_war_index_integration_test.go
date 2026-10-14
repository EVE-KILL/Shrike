package api

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type factionWarPlanDB struct {
	Database
	sql  string
	args []any
}

func (db *factionWarPlanDB) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	db.sql, db.args = query, args
	return db.Database.Query(ctx, query, args...)
}

func TestFactionWarValueListUsesRecentFactionRangeAndPreservesRanking(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_faction_range_")
	_, err := pool.Exec(ctx, `
        INSERT INTO inv_types(type_id,name) VALUES(587,'Rifter');
        INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_ship_type_id,total_value,fw_winner_faction_id)
        SELECT 1000+n,'old',now()-interval '100 days'-n*interval '1 minute',30000142,587,1000000000,
               CASE WHEN n%50=0 THEN 500001 ELSE NULL END
        FROM generate_series(1,50000)n;
        INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_ship_type_id,total_value,fw_winner_faction_id)
        SELECT 100000+n,'other',now()-interval '1 day',30000142,587,1000000000,500004
        FROM generate_series(1,20000)n;
        INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_ship_type_id,total_value,fw_winner_faction_id)
        SELECT 100+n,'target',now()-interval '1 day',30000142,
               CASE WHEN n=12 THEN 999999 ELSE 587 END,
               CASE WHEN n>=10 THEN 100 ELSE n END,500001
        FROM generate_series(1,12)n;`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `VACUUM ANALYZE killmails`); err != nil {
		t.Fatal(err)
	}
	db := &factionWarPlanDB{Database: pool}
	rows, err := loadMostValuableKills(ctx, db, "fw-caldari-winner", 7, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 8 {
		t.Fatalf("got %d rows, want 8", len(rows))
	}
	for i, row := range rows {
		if int64OrZero(row["killmail_id"]) != int64(111-i) || row["ship_name"] != "Rifter" {
			t.Fatalf("ranking, date/faction scope or valid-type selection changed at %d: %v", i, row)
		}
	}
	var plan []byte
	if err := pool.QueryRow(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+db.sql, db.args...).Scan(&plan); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plan), `killmails_fw_winner_time_idx`) {
		t.Fatal("recent faction selection did not use killmails_fw_winner_time_idx")
	}
}
