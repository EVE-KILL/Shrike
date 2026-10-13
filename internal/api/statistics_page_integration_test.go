package api

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestStatisticsReaderScopeOrderingAndWarmBoardsAgainstPostgres(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_statistics_test_")
	_, err := pool.Exec(ctx, `
		INSERT INTO users (character_id,character_name,character_owner_hash,session_id) VALUES (1,'Fixture owner','fixture','00000000-0000-4000-8000-000000000001');
		INSERT INTO characters (character_id,name,security_status,birthday) VALUES
		 (900000001,'First pilot',6,'2026-09-23T12:00:00Z'),
		 (900000002,'Second pilot',6,'2026-09-23T12:00:00Z');
		INSERT INTO corporations (corporation_id,name,ticker,date_founded) VALUES (9001001,'Fixture corp','FIX','2026-09-23T12:00:00Z');
		INSERT INTO entity_snapshots (entity_type,entity_id,date,member_count,avg_sec_status,total_achievement_points,avg_achievement_points)
		 VALUES ('corporation',9001001,current_date,100,2.5,1000,10),('corporation',9001001,current_date-7,50,2.5,1000,10);
		INSERT INTO stats (entity_type,entity_id,period_type,period_start,kills,isk_destroyed) VALUES
		 (0,900000001,0,current_date,100,2000000000),(0,900000002,0,current_date,200,3000000000);
		INSERT INTO custom_domains (subdomain,user_id,site_name,entities)
		 VALUES ('stats-fixture',1,'Fixture board','[{"type":"character","id":900000001,"name":"First pilot"}]');
	`)
	if err != nil {
		t.Fatal(err)
	}
	db := &campaignCreatorTestDB{pool}
	service := New(Options{DB: db})
	reader := newCachedHomepageReader(db, nil, NewResponseCache(nil, 1<<20), "statistics-test")
	global, err := reader.StatisticsSite(ctx, "localhost:4200")
	if err != nil || global.Domain != nil {
		t.Fatal("loopback host did not resolve globally")
	}
	domain, err := reader.StatisticsSite(ctx, "stats-fixture.localhost:4200")
	if err != nil || domain.Domain == nil {
		t.Fatal("the normalized local domain did not resolve")
	}
	public := func(host, path string) []StatisticsEntry {
		t.Helper()
		w := httptest.NewRecorder()
		service.Site().ServeHTTP(w, httptest.NewRequest("GET", "http://"+host+path, nil))
		if w.Code != 200 {
			t.Fatalf("stats API: %d %s", w.Code, w.Body.String())
		}
		var body struct {
			Entries []StatisticsEntry `json:"entries"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Entries
	}
	queries := []StatisticsQuery{{DataType: "characters", Days: 7}, {DataType: "carebear_characters", Days: 7}, {Section: "newest", EntityType: "character", Days: 7}, {Section: "largest", EntityType: "corporation", Days: 7}}
	paths := []string{"/api/stats?dataType=characters&days=7", "/api/stats?dataType=carebear_characters", "/api/stats/rankings?section=newest&entityType=character", "/api/stats/rankings?section=largest&entityType=corporation"}
	if len(reader.StatisticsBoardsCached(ctx, global, queries)) != 0 {
		t.Fatal("cold boards were reported as loaded")
	}
	for i, query := range queries {
		rows, err := reader.StatisticsBoard(ctx, global, query)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rows, public("localhost", paths[i])) {
			t.Fatalf("shared card %d and public API differ", i)
		}
		if i == 1 && (len(rows) != 2 || rows[0].ID != 900000001) {
			t.Fatal("security ties were not deterministic")
		}
		if i == 2 && (len(rows) != 2 || rows[0].ID != 900000002) {
			t.Fatal("birth timestamp ties were not deterministic")
		}
	}
	if len(reader.StatisticsBoardsCached(ctx, global, queries)) != 4 {
		t.Fatal("the page could not reuse its loaded boards")
	}
	query := StatisticsQuery{DataType: "characters", Days: 7}
	rows, err := reader.StatisticsBoard(ctx, domain, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != 900000001 || !reflect.DeepEqual(rows, public("stats-fixture.localhost", "/api/custom/stats?dataType=characters&days=7")) {
		t.Fatal("domain card leaked the global population or diverged from its API")
	}
	if _, err := reader.StatisticsBoard(ctx, domain, queries[3]); err == nil {
		t.Fatal("domain requested a global population board")
	}
	changed := domain
	copyDomain := *domain.Domain
	copyDomain.EntityIDs.CharacterIDs = []int32{900000002}
	changed.Domain = &copyDomain
	if len(reader.StatisticsBoardsCached(ctx, changed, []StatisticsQuery{query})) != 0 {
		t.Fatal("changed entity scope reused the previous domain's cache")
	}
}
