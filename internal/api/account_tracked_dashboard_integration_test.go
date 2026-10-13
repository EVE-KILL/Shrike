package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/testfixtures"
)

func TestTrackedDashboardDeduplicatesScopeFiltersAndIsolatesConfiguration(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_tracked_dashboard_")
	if err := testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id) VALUES(42,'Owner','fixture','00000000-0000-4000-8000-000000000042'),(43,'Reader','fixture','00000000-0000-4000-8000-000000000043')`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
	call := func(method, path, body, account string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		r.Header.Set("Origin", "http://localhost")
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		if account != "" {
			r.AddCookie(&http.Cookie{Name: "evelogin", Value: "00000000-0000-4000-8000-0000000000" + account})
		}
		w := httptest.NewRecorder()
		service.Site().ServeHTTP(w, r)
		return w
	}
	for _, target := range []string{`{"target_type":"corporation","target_id":70111}`, `{"target_type":"system","target_id":30000142}`} {
		if w := call("POST", "/api/me/trackers", target, "42"); w.Code != 201 {
			t.Fatalf("tracker=%d", w.Code)
		}
	}
	_, err := pool.Exec(ctx, `INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,region_id,victim_character_id,victim_corporation_id,victim_alliance_id,victim_ship_type_id,victim_ship_group_id,total_value,attacker_count,is_npc,is_solo) SELECT 2100000000+i,'tracked-dashboard-'||i,now(),solar_system_id,region_id,victim_character_id,victim_corporation_id,victim_alliance_id,victim_ship_type_id,victim_ship_group_id,i::bigint*1000000000,attacker_count,i=1,i=2 FROM killmails CROSS JOIN generate_series(1,3)i WHERE killmail_id=2000000002`)
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`UPDATE inv_groups SET category_id=6 WHERE group_id IN (25,420)`,
		`INSERT INTO killmail_attackers(killmail_id,attacker_index,character_id,corporation_id,alliance_id,faction_id,ship_type_id,ship_group_id,damage_done,final_blow,killmail_time) SELECT 2100000000+i,attacker_index,character_id,corporation_id,alliance_id,faction_id,ship_type_id,ship_group_id,damage_done,final_blow,now() FROM killmail_attackers CROSS JOIN generate_series(1,3)i WHERE killmail_id=2000000002`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	var events int
	pool.QueryRow(ctx, "SELECT count(*) FROM entity_tracker_events").Scan(&events)
	if events != 6 {
		t.Fatalf("overlapping watches recorded %d events", events)
	}
	var summary struct {
		Kills int     `json:"kill_count"`
		Value float64 `json:"total_value"`
	}
	w := call("GET", "/api/me/tracked/summary?days=7", "", "42")
	json.Unmarshal(w.Body.Bytes(), &summary)
	if w.Code != 200 || summary.Kills != 3 || summary.Value != 6e9 {
		t.Fatalf("summary doubled overlapping kills: %d %#v", w.Code, summary)
	}
	for _, tc := range []struct {
		kind  string
		count int
	}{{"latest", 3}, {"big", 3}, {"solo", 1}, {"npc", 1}} {
		var page KilllistPage
		w := call("GET", "/api/me/tracked/killmails?type="+tc.kind, "", "42")
		json.Unmarshal(w.Body.Bytes(), &page)
		if w.Code != 200 || len(page.Kills) != tc.count {
			t.Fatalf("%s=%d rows %d", tc.kind, w.Code, len(page.Kills))
		}
	}
	for _, kind := range []string{"ships", "systems", "regions", "most_valuable_kills", "most_valuable_ships"} {
		w := call("GET", "/api/me/tracked/stats?dataType="+kind, "", "42")
		var data struct {
			Entries []map[string]any `json:"entries"`
		}
		json.Unmarshal(w.Body.Bytes(), &data)
		if w.Code != 200 || len(data.Entries) == 0 {
			t.Fatalf("%s=%d entries=%d", kind, w.Code, len(data.Entries))
		}
	}
	body := `{"widgets":{"top":[{"type":"textBlock","enabled":true,"content":"Private note"}],"left":[],"right":[{"type":"killList","enabled":true,"killlistType":"npc"}],"columnRatio":"1fr_2fr"}}`
	if w := call("PUT", "/api/me/tracked/config", body, "42"); w.Code != 200 {
		t.Fatalf("save=%d %s", w.Code, w.Body.String())
	}
	var config struct {
		Configured bool              `json:"configured"`
		Widgets    SiteDomainWidgets `json:"widgets"`
	}
	w = call("GET", "/api/me/tracked/config", "", "42")
	json.Unmarshal(w.Body.Bytes(), &config)
	if !config.Configured || config.Widgets.Top[0].Content == nil || *config.Widgets.Top[0].Content != "Private note" || *config.Widgets.Right[0].KilllistType != "npc" {
		t.Fatal("configuration lost content or feed type")
	}
	for _, path := range []string{"/api/me/tracked/config", "/api/me/tracked/summary", "/api/me/tracked/killmails", "/api/me/tracked/stats?dataType=systems"} {
		reader := call("GET", path, "", "43")
		if reader.Code != 200 || strings.Contains(reader.Body.String(), "Private note") || strings.Contains(reader.Body.String(), "210000000") {
			t.Fatal("private dashboard data crossed accounts")
		}
		if guest := call("GET", path, "", ""); guest.Code != 401 {
			t.Fatal("anonymous dashboard access allowed")
		}
	}
	var readerConfig struct {
		Configured bool `json:"configured"`
	}
	json.Unmarshal(call("GET", "/api/me/tracked/config", "", "43").Body.Bytes(), &readerConfig)
	if readerConfig.Configured {
		t.Fatal("reader inherited owner's layout")
	}
	if w := call("PUT", "/api/me/tracked/config", `{"widgets":{"top":[],"left":[],"right":[]}}`, "42"); w.Code != 400 {
		t.Fatal("empty dashboard accepted")
	}
}
