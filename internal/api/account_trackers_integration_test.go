package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/testfixtures"
)

func TestTrackersRecordFutureActivityPauseAlertsCursorsAndOwnership(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_trackers_")
	if err := testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id) VALUES(42,'Owner','fixture','00000000-0000-4000-8000-000000000042'),(43,'Reader','fixture','00000000-0000-4000-8000-000000000043')`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
	} {
		if _, err := pool.Exec(ctx, query); err != nil {
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
	created := call("POST", "/api/me/trackers", `{"target_type":"corporation","target_id":70111}`, "42")
	var tracker struct {
		ID            int64 `json:"id"`
		Notifications bool  `json:"notifications_enabled"`
		EventCount    int64 `json:"event_count"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &tracker); err != nil {
		t.Fatal(err)
	}
	if created.Code != 201 || tracker.ID <= 0 || tracker.Notifications || tracker.EventCount != 0 {
		t.Fatalf("creation backfilled history or enabled alerts: %d %#v", created.Code, tracker)
	}
	base := "/api/me/trackers/" + strconv.FormatInt(tracker.ID, 10)
	if response := call("POST", "/api/me/trackers", `{"target_type":"corporation","target_id":70111}`, "42"); response.Code != 409 {
		t.Fatalf("duplicate tracker=%d", response.Code)
	}
	insert := func(start, count int) {
		t.Helper()
		_, err := pool.Exec(ctx, `INSERT INTO killmails(killmail_id,killmail_hash,killmail_time,solar_system_id,region_id,victim_character_id,victim_corporation_id,victim_alliance_id,victim_ship_type_id,victim_ship_group_id,total_value,attacker_count) SELECT $1+i,'tracker-fixture-'||($1+i),now(),solar_system_id,region_id,victim_character_id,victim_corporation_id,victim_alliance_id,victim_ship_type_id,victim_ship_group_id,total_value,attacker_count FROM killmails CROSS JOIN generate_series(0,$2-1) i WHERE killmail_id=2000000002`, start, count)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert(2100000001, 27)
	var events, alerts int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM entity_tracker_events WHERE tracker_id=$1", tracker.ID).Scan(&events)
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM entity_tracker_notifications").Scan(&alerts)
	if events != 27 || alerts != 0 {
		t.Fatalf("recording and alert opt-in diverged: events=%d alerts=%d", events, alerts)
	}
	for _, field := range []string{`{"enabled":false}`, `{"enabled":true,"notifications_enabled":true}`} {
		if response := call("PATCH", base, field, "42"); response.Code != 200 {
			t.Fatalf("update=%d %s", response.Code, response.Body.String())
		}
		if strings.Contains(field, ":false") {
			insert(2100000030, 1)
		}
	}
	insert(2100000040, 1)
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM entity_tracker_events WHERE tracker_id=$1", tracker.ID).Scan(&events)
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM entity_tracker_notifications").Scan(&alerts)
	if events != 28 || alerts != 1 {
		t.Fatalf("paused/resumed opt-in events=%d alerts=%d", events, alerts)
	}
	var page KilllistPage
	response := call("GET", base+"/killmails?limit=10", "", "42")
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || len(page.Kills) != 10 || !page.HasMore || page.Cursor == nil || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("first cursor page=%d rows=%d", response.Code, len(page.Kills))
	}
	first := page.Kills[0].KillmailID
	cursor := *page.Cursor
	response = call("GET", base+"/killmails?limit=10&after="+strconv.FormatInt(cursor, 10), "", "42")
	_ = json.Unmarshal(response.Body.Bytes(), &page)
	if response.Code != 200 || len(page.Kills) != 10 || page.Kills[0].KillmailID >= cursor || first != 2100000040 {
		t.Fatal("cursor repeated or reordered recorded kills")
	}
	for _, operation := range []struct{ Method, Path, Body string }{{"GET", base + "/killmails", ""}, {"PATCH", base, `{"enabled":false}`}, {"DELETE", base, ""}} {
		if response := call(operation.Method, operation.Path, operation.Body, "43"); response.Code != 404 {
			t.Errorf("foreign account %s=%d", operation.Method, response.Code)
		}
	}
	if response := call("GET", base+"/killmails", "", ""); response.Code != 401 {
		t.Fatalf("anonymous feed=%d", response.Code)
	}
	if response := call("DELETE", base, "", "42"); response.Code != 200 {
		t.Fatalf("delete=%d", response.Code)
	}
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM entity_tracker_events").Scan(&events)
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM entity_tracker_notifications").Scan(&alerts)
	if events != 0 || alerts != 0 {
		t.Fatal("tracker deletion left activity or alerts behind")
	}
}
