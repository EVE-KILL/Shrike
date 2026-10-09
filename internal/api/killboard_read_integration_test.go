package api

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/jackc/pgx/v5"
)

func TestValueBandKilllistPagingRetainsBoundariesTiesAndStaleDayCaps(t *testing.T) {
	start := time.Now().UTC().Truncate(24 * time.Hour)
	for index, band := range []struct {
		kind      string
		low, high int64
	}{
		{"5b-10b", 5_000_000_000, 10_000_000_000},
		{"10b-100b", 10_000_000_000, 100_000_000_000},
	} {
		ctx, pool := localAPITestPool(t, "shrike_killboard_value_")
		base := int64(index+1) * 10000
		for day, count := range []int{3, 1, 2} { // Deliberately overcount/undercount the first two days.
			date := start.AddDate(0, 0, -day)
			if _, err := pool.Exec(ctx, `INSERT INTO kills_daily_count(date,type,count) VALUES($1,$2,$3)`, date, band.kind, count); err != nil {
				t.Fatal(err)
			}
			for entry, value := range []int64{band.low - 1, band.low, band.high - 1, band.high} {
				id := base + int64(day+1)*100 + int64(entry)
				if _, err := pool.Exec(ctx, `INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_ship_type_id,total_value) VALUES($1,'value-band',$2,30000142,587,$3)`, id, date.Add(time.Hour), value); err != nil {
					t.Fatal(err)
				}
			}
		}
		for _, test := range []struct {
			page, limit int
			ids         []int64
			more        bool
		}{
			{1, 4, []int64{base + 102, base + 101, base + 202, base + 302}, true},
			{2, 4, []int64{base + 302, base + 301}, false},
			{1, 6, []int64{base + 102, base + 101, base + 202, base + 302, base + 301}, false},
		} {
			payload, err := loadNumberedKilllist(ctx, pool, band.kind, killtype.Predicates()[band.kind], test.page, test.limit)
			if err != nil {
				t.Fatal(err)
			}
			body := payload.Body.(map[string]any)
			rows := body["kills"].([]map[string]any)
			ids := make([]int64, len(rows))
			for i, row := range rows {
				ids[i] = int64OrZero(row["killmail_id"])
				if value := float64OrZero(row["total_value"]); value < float64(band.low) || value >= float64(band.high) || row["killmail_hash"] != "value-band" {
					t.Fatalf("%s lost its boundaries or display data: %v", band.kind, row)
				}
			}
			if !slices.Equal(ids, test.ids) || body["hasMore"] != test.more {
				t.Fatalf("%s page %d limit %d: ids=%v more=%v", band.kind, test.page, test.limit, ids, body["hasMore"])
			}
		}
	}
}

func TestKillboardBoundedReadsRetainValidTypesBandsTiesAndDayOrdering(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_killboard_read_")
	db := &campaignCreatorTestDB{pool}
	if _, err := pool.Exec(ctx, `INSERT INTO inv_types(type_id,name,group_id) VALUES(587,'Rifter',25)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO inv_groups(group_id,name,category_id) VALUES(25,'Frigate',6)`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	for _, kill := range []struct {
		id, ship int64
		value    float64
		age      time.Duration
	}{
		{101, 587, 9e9, time.Minute}, {102, 587, 9e9, 2 * time.Minute},
		{103, 587, 8e9, 3 * time.Minute}, {104, 999999, 9.5e9, time.Minute},
		{105, 587, 10e9, time.Minute}, {106, 587, 4.9e9, time.Minute},
		{107, 587, 7e9, 9 * 24 * time.Hour},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_ship_type_id,victim_ship_group_id,total_value) VALUES($1,'test',$2,30000142,$3,25,$4)`, kill.id, now.Add(-kill.age), kill.ship, kill.value); err != nil {
			t.Fatal(err)
		}
	}
	valuable, err := loadMostValuableKills(ctx, db, "5b-10b", 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(valuable) != 3 {
		t.Fatalf("unknown type consumed a result slot: %v", valuable)
	}
	for index, id := range []int64{102, 101, 103} {
		if int64OrZero(valuable[index]["killmail_id"]) != id || valuable[index]["ship_name"] != "Rifter" {
			t.Fatalf("value band, stable tie or enrichment changed: %v", valuable)
		}
	}
	ships, err := loadMostValuableKills(ctx, db, "category-ship", 7, 2)
	if err != nil || len(ships) != 2 || int64OrZero(ships[0]["killmail_id"]) != 105 || int64OrZero(ships[1]["killmail_id"]) != 102 {
		t.Fatalf("recent category ranking changed: %v %v", ships, err)
	}
	// A newer kill can have a smaller ID. Day paging follows time, then ID.
	start := now.Add(-24 * time.Hour)
	if _, err := pool.Exec(ctx, `INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_ship_type_id,total_value) VALUES(299,'test',$1,30000142,587,1),(310,'test',$2,30000142,587,1),(300,'test',$3,30000142,587,1)`, start.Add(time.Minute), start.Add(2*time.Minute), start.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	rows, err := loadKilllistDay(ctx, db, "k.total_value=1", start, 1, 1)
	if err != nil || len(rows) != 1 || int64OrZero(rows[0]["killmail_id"]) != 310 || rows[0]["ship_name"] != "Rifter" {
		t.Fatalf("day offset/order/enrichment changed: %v %v", rows, err)
	}
}

type killboardCountingDB struct {
	Database
	queries int
}

func (db *killboardCountingDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	db.queries++
	return db.Database.Query(ctx, sql, args...)
}

func TestNumberedKilllistBatchesSparseDaysAndPreservesStaleRollupPaging(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_killboard_sparse_")
	db := &killboardCountingDB{Database: pool}
	start := time.Now().UTC().Truncate(24 * time.Hour)
	seedDay := func(kind string, day, count int, ids []int64, value int) {
		t.Helper()
		date := start.AddDate(0, 0, -day)
		if _, err := pool.Exec(ctx, `INSERT INTO kills_daily_count(date,type,count) VALUES($1,$2,$3)`, date, kind, count); err != nil {
			t.Fatal(err)
		}
		for index, id := range ids {
			// Include tied timestamps: the ID is the stable second sort key.
			if _, err := pool.Exec(ctx, `INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,victim_ship_type_id,total_value) VALUES($1,'test',$2,30000142,587,$3)`, id, date.Add(time.Hour+time.Duration(index/2)*time.Minute), value); err != nil {
				t.Fatal(err)
			}
		}
	}
	for day := range 50 {
		seedDay("sparse-test", day, 1, []int64{int64(1000 - day)}, 1)
	}
	page, err := loadNumberedKilllist(ctx, db, "sparse-test", "k.total_value=1", 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	rows := page.Body.(map[string]any)["kills"].([]map[string]any)
	if len(rows) != 50 || db.queries != 2 {
		t.Fatalf("50 sparse days require one rollup and one page read: rows=%d queries=%d", len(rows), db.queries)
	}
	for index, row := range rows {
		if int64OrZero(row["killmail_id"]) != int64(1000-index) {
			t.Fatalf("sparse day order changed at %d: %v", index, row)
		}
	}

	seedDay("stale-test", 0, 4, []int64{101, 102}, 2)      // Overcounted.
	seedDay("stale-test", 1, 2, []int64{201, 202, 203}, 2) // Undercounted: retain the rollup cap.
	seedDay("stale-test", 2, 1, nil, 2)                    // Empty despite its rollup.
	seedDay("stale-test", 3, 3, []int64{301, 302, 303}, 2)
	seedDay("stale-test", 4, 2, []int64{401, 402}, 2)
	seedDay("stale-test", 5, 1, []int64{501}, 2)
	for _, test := range []struct {
		page, limit int
		ids         []int64
		hasMore     bool
	}{
		{1, 5, []int64{102, 101, 203, 202, 303}, true},
		{2, 5, []int64{202, 303, 302, 301, 402}, true},
		{3, 5, []int64{402, 401, 501}, false},
		{4, 5, []int64{}, false},
		{1, 1, []int64{102}, true},
		{5, 1, []int64{203}, true},
	} {
		page, err := loadNumberedKilllist(ctx, db, "stale-test", "k.total_value=2", test.page, test.limit)
		if err != nil {
			t.Fatal(err)
		}
		body := page.Body.(map[string]any)
		rows := body["kills"].([]map[string]any)
		ids := make([]int64, len(rows))
		for index, row := range rows {
			ids[index] = int64OrZero(row["killmail_id"])
		}
		if !slices.Equal(ids, test.ids) || body["hasMore"] != test.hasMore {
			t.Fatalf("page %d limit %d changed rollup semantics: ids=%v hasMore=%v", test.page, test.limit, ids, body["hasMore"])
		}
	}
}
