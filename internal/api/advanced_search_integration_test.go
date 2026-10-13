package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAdvancedSearchFilteringCursorAndFitsAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	var random [6]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	name := "shrike_searchtest_" + hex.EncodeToString(random[:])
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Close()
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("clean search test database: %v", err)
		}
	}()
	if err := migrate.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO corporations (corporation_id,name,ticker) VALUES (10,'Fleet','FLT'),(11,'Targets','TGT')`,
		`INSERT INTO characters (character_id,name,corporation_id) VALUES (1,'Alpha',10),(2,'Beta',10),(3,'Victim',11)`,
		`INSERT INTO regions (region_id,name) VALUES (100,'Test Region')`,
		`INSERT INTO solar_systems (solar_system_id,system_name,region_id,security) VALUES (30000142,'Test System',100,0.9)`,
		`INSERT INTO inv_groups (group_id,name,category_id) VALUES (25,'Frigate',6),(55,'Projectile Weapon',7),(100,'Combat Drone',18)`,
		`INSERT INTO inv_types (type_id,name,group_id,meta_group_id) VALUES (587,'Rifter',25,1),(425,'Autocannon',55,1),(2456,'Hobgoblin',100,1)`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	end := time.Now().UTC().Truncate(time.Second)
	start := end.Add(-time.Hour)
	for _, kill := range []struct {
		id      int
		minutes int
		value   float64
		count   int
	}{{101, 10, 10e6, 1}, {102, 20, 100e6, 1}, {103, 20, 500e6, 2}, {104, 30, 0, 1}} {
		at := start.Add(time.Duration(kill.minutes) * time.Minute)
		_, err := pool.Exec(ctx, `INSERT INTO killmails (killmail_id,killmail_hash,killmail_time,solar_system_id,region_id,victim_character_id,victim_corporation_id,victim_ship_type_id,victim_ship_group_id,total_value,attacker_count,is_solo) VALUES ($1,'test',$2,30000142,100,3,11,587,25,$3,$4,$5)`, kill.id, at, kill.value, kill.count, kill.count == 1)
		if err != nil {
			t.Fatal(err)
		}
		actor := 1
		if kill.id == 102 {
			actor = 2
		}
		if _, err := pool.Exec(ctx, `INSERT INTO killmail_attackers (killmail_id,attacker_index,character_id,corporation_id,ship_type_id,weapon_type_id,killmail_time,final_blow) VALUES ($1,0,$2,10,587,425,$3,true)`, kill.id, actor, at); err != nil {
			t.Fatal(err)
		}
		if kill.id == 103 {
			if _, err := pool.Exec(ctx, `INSERT INTO killmail_attackers (killmail_id,attacker_index,character_id,corporation_id,ship_type_id,weapon_type_id,killmail_time) VALUES (103,1,2,10,587,425,$1)`, at); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := pool.Exec(ctx, `INSERT INTO killmail_items (killmail_id,item_index,type_id,flag_id,quantity_destroyed) VALUES ($1,0,425,27,1),($1,1,2456,87,7)`, kill.id); err != nil {
			t.Fatal(err)
		}
	}
	base := map[string]any{"entities": map[string]any{"attacker": []map[string]any{{"id": 1, "type": "character"}}}, "timeRange": map[string]any{"from": start.Format(time.RFC3339), "to": end.Format(time.RFC3339)}}
	reader := NewHomepageReader(pool)
	search := func(filters map[string]any, extra url.Values) AdvancedSearchResult {
		t.Helper()
		encoded, _ := json.Marshal(filters)
		query := url.Values{"filters": {string(encoded)}, "limit": {"2"}}
		for key, values := range extra {
			query[key] = values
		}
		result, err := reader.AdvancedSearch(ctx, query)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	ids := func(rows []KilllistRow) []int64 {
		out := []int64{}
		for _, row := range rows {
			out = append(out, row.KillmailID)
		}
		return out
	}
	first := search(base, nil)
	if !reflect.DeepEqual(ids(first.Kills), []int64{104, 103}) || !first.HasMore || first.Cursor == nil || *first.Cursor != 103 {
		t.Fatalf("date/tie/cursor first page: %+v", first)
	}
	second := search(base, url.Values{"after": {"103"}})
	if !reflect.DeepEqual(ids(second.Kills), []int64{101}) || second.HasMore {
		t.Fatalf("cursor second page: %+v", second)
	}
	sorted := map[string]any{}
	for key, value := range base {
		sorted[key] = value
	}
	sorted["sort"] = map[string]any{"field": "total_value", "direction": "asc"}
	if got := search(sorted, nil); !reflect.DeepEqual(ids(got.Kills), []int64{104, 101}) {
		t.Fatalf("value ordering: %+v", got)
	}
	if got := search(sorted, url.Values{"after": {"101"}}); !reflect.DeepEqual(ids(got.Kills), []int64{103}) {
		t.Fatalf("value cursor: %+v", got)
	}
	filtered := map[string]any{}
	for key, value := range base {
		filtered[key] = value
	}
	filtered["entities"] = map[string]any{"attacker": []map[string]any{{"id": 1, "type": "character"}, {"id": 2, "type": "character", "exclude": true}}}
	filtered["location"] = map[string]any{"systemId": 30000142, "securityTypes": []string{"highsec"}}
	filtered["items"] = []map[string]any{{"groupId": 55, "slot": "fitted", "side": "victim"}}
	if got := search(filtered, nil); !reflect.DeepEqual(ids(got.Kills), []int64{104, 101}) {
		t.Fatalf("entity exclusion/location/item group: %+v", got)
	}
	fits := search(base, url.Values{"view": {"fits"}, "dedup": {"none"}})
	if len(fits.Fits) != 2 || fits.Fits[0].KillmailID != 104 || len(fits.Fits[0].Modules) != 1 || fits.Fits[0].Modules[0].TypeID != 425 || len(fits.Fits[0].Drones) != 1 || fits.Fits[0].Drones[0].Quantity != 7 || !fits.HasMore {
		t.Fatalf("bounded fit projection/modules/drones: %+v", fits)
	}
}
