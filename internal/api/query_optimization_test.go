package api

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/battle"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type queryOptimizationTx struct{ pgx.Tx }

func (db queryOptimizationTx) Ping(ctx context.Context) error {
	_, err := db.Exec(ctx, "SELECT 1")
	return err
}

func TestBattleParticipantCountsUseDisplayedKills(t *testing.T) {
	kills := []battle.Killmail{{KillmailID: 1}, {KillmailID: 2}, {KillmailID: 1}}
	attackers := map[int64][]battle.Attacker{
		1: {{CharacterID: 10, CorporationID: 20, AllianceID: 30, DamageDone: 50}, {CorporationID: 21, DamageDone: 25}},
		2: {{CharacterID: 10, CorporationID: 20, AllianceID: 30, DamageDone: 100}, {CharacterID: 11, CorporationID: 22, DamageDone: 10}},
		3: {{CharacterID: 99, CorporationID: 99, AllianceID: 99, DamageDone: 1000}},
	}
	want := map[string]any{"characters": 2, "corporations": 3, "alliances": 1, "total_damage": int64(185)}
	if got := battleParticipantCounts(kills, attackers); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if got := battleParticipantCounts(nil, attackers); got["total_damage"] != int64(0) || got["characters"] != 0 {
		t.Fatal(got)
	}
}

func TestNearestPricesPreserveHistoryAndOverrides(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires explicit test database")
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
	db := queryOptimizationTx{tx}
	_, err = tx.Exec(ctx, `CREATE TEMP TABLE prices(type_id int,region_id int,date date,average double precision,PRIMARY KEY(type_id,region_id,date)) ON COMMIT DROP;
        CREATE TEMP TABLE custom_prices(type_id int,date date,price double precision,PRIMARY KEY(type_id,date)) ON COMMIT DROP;
        INSERT INTO prices VALUES (1,10000002,'2026-09-04',40),(1,10000002,'2026-09-06',60),
        (1,9,'2026-09-05',999),(2,10000002,'2026-09-01',20), (3,10000002,'2026-09-10',30),
        (4,10000002,'2026-09-05',NULL), (5,10000002,'2026-09-05',50);
        INSERT INTO custom_prices VALUES (5,'2026-09-04',54),(5,'2026-09-06',56),(6,'2026-09-05',0);`)
	if err != nil {
		t.Fatal(err)
	}
	ids := []int32{1, 2, 3, 4, 5, 6, 7, 1}
	day := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	for _, spec := range []struct{ table, value, region string }{{"prices", "average", "AND region_id=10000002"}, {"custom_prices", "price", ""}} {
		old, err := queryMaps(ctx, db, "SELECT DISTINCT ON(type_id) type_id,"+spec.value+" FROM "+spec.table+" WHERE type_id=ANY($1::int[]) "+spec.region+" ORDER BY type_id,ABS(date-$2::date),date DESC", ids, day)
		if err != nil {
			t.Fatal(err)
		}
		got, err := queryMaps(ctx, db, nearestTypePricesSQL(spec.table, spec.value, spec.region)+" ORDER BY requested.type_id", ids, day)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(old, got) {
			t.Fatalf("%s: old=%v new=%v", spec.table, old, got)
		}
	}
	got, err := loadTypePrices(ctx, db, ids, day)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int64]float64{1: 60, 2: 20, 3: 30, 4: 0, 5: 56, 6: 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
