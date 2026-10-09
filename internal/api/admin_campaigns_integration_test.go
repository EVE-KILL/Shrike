package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminCampaignPaginationFiltersActionsAndFundedGuards(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_admin_campaigns_")
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture admin','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture reader','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
		`INSERT INTO campaigns(campaign_id,name,created_by_character_id,status,visibility,start_time,created_at,processing_paused,last_processing_error) SELECT 'AdminCamp'||to_char(i,'FM00000'),CASE WHEN i=1 THEN 'Literal 100%_ match' ELSE 'Campaign '||i END,CASE WHEN i%2=0 THEN 43 ELSE 42 END,i%3,i%3,now()-interval '1 day',now(),i%7=0,CASE WHEN i%11=0 THEN 'Processing failed' ELSE NULL END FROM generate_series(1,56)i`,
		`INSERT INTO campaign_prize_pools(campaign_id,metric,winner_count,payout_percentages,funded_total) VALUES('AdminCamp00001',0,3,'{70,20,10}',100)`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
	call := func(method, path, body, identity string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		r.Header.Set("Origin", "http://localhost")
		r.Header.Set("Content-Type", "application/json")
		if identity != "" {
			r.AddCookie(&http.Cookie{Name: "evelogin", Value: "00000000-0000-4000-8000-0000000000" + identity})
		}
		w := httptest.NewRecorder()
		service.Site().ServeHTTP(w, r)
		return w
	}
	for _, identity := range []string{"", "43"} {
		want := 401
		if identity == "43" {
			want = 403
		}
		for _, route := range []struct{ Method, Path, Body string }{{"GET", "/api/admin/campaigns", ""}, {"POST", "/api/admin/campaigns/AdminCamp00002/action", `{"action":"pause"}`}} {
			w := call(route.Method, route.Path, route.Body, identity)
			if w.Code != want || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatal("unauthorized campaign administration")
			}
		}
	}
	type row struct {
		ID      string  `json:"campaign_id"`
		Status  int     `json:"status"`
		Paused  bool    `json:"processing_paused"`
		Error   *string `json:"last_processing_error"`
		Creator string  `json:"creator_name"`
	}
	list := func(query string) ([]row, int, bool) {
		w := call("GET", "/api/admin/campaigns?"+query, "", "42")
		if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
			t.Fatalf("list=%d", w.Code)
		}
		var data struct {
			Campaigns []row `json:"campaigns"`
			Page      int   `json:"page"`
			More      bool  `json:"hasMore"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		return data.Campaigns, data.Page, data.More
	}
	first, _, more := list("page=1&limit=100000")
	second, page, more2 := list("page=2")
	if len(first) != 40 || !more || len(second) != 16 || page != 2 || more2 {
		t.Fatal("bounded campaign pages diverged")
	}
	seen := map[string]bool{}
	for _, r := range append(first, second...) {
		if seen[r.ID] {
			t.Fatal("equal timestamps duplicated a campaign across pages")
		}
		seen[r.ID] = true
	}
	again, _, _ := list("page=1")
	for i, r := range first {
		if r.ID != again[i].ID {
			t.Fatal("campaign paging order changed")
		}
	}
	for state, want := range map[string]int{"paused": 8, "failed": 5, "pending": 18, "active": 19, "archived": 19} {
		rows, _, _ := list("state=" + state)
		if len(rows) != want {
			t.Fatalf("%s count=%d", state, len(rows))
		}
		for _, r := range rows {
			if state == "paused" && !r.Paused || state == "failed" && r.Error == nil || state == "pending" && r.Status != 0 || state == "active" && r.Status != 1 || state == "archived" && r.Status != 2 {
				t.Fatal("state filter leaked rows")
			}
		}
	}
	rows, _, _ := list("q=100%25_")
	if len(rows) != 1 || rows[0].ID != "AdminCamp00001" {
		t.Fatal("search wildcards were not literal")
	}
	rows, _, _ = list("q=Fixture%20reader&state=active")
	if len(rows) != 9 {
		t.Fatalf("creator/status filters=%d", len(rows))
	}
	for _, action := range []string{"pause", "resume", "reprocess", "archive"} {
		w := call("POST", "/api/admin/campaigns/AdminCamp00002/action", fmt.Sprintf(`{"action":%q}`, action), "42")
		if w.Code != 200 {
			t.Fatalf("%s=%d", action, w.Code)
		}
		var status int
		var paused bool
		var note, processingError *string
		if err := pool.QueryRow(ctx, `SELECT status,processing_paused,processing_note,last_processing_error FROM campaigns WHERE campaign_id='AdminCamp00002'`).Scan(&status, &paused, &note, &processingError); err != nil {
			t.Fatal(err)
		}
		switch action {
		case "pause":
			if !paused || note == nil || *note != "Paused by Fixture admin" {
				t.Fatal("pause did not persist")
			}
		case "resume", "reprocess":
			if paused || status != 0 || note != nil || processingError != nil {
				t.Fatal("processing restart did not reset state")
			}
		case "archive":
			if !paused || status != 2 {
				t.Fatal("archive did not persist")
			}
		}
	}
	for _, action := range []string{"archive", "delete"} {
		if w := call("POST", "/api/admin/campaigns/AdminCamp00001/action", fmt.Sprintf(`{"action":%q}`, action), "42"); w.Code != 400 {
			t.Fatal("funded prize guard failed")
		}
	}
	if w := call("POST", "/api/admin/campaigns/AdminCamp00002/actions", `{"action":"delete"}`, "42"); w.Code != 200 {
		t.Fatal("canonical delete action failed")
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM campaigns WHERE campaign_id='AdminCamp00002')`).Scan(&exists); err != nil || exists {
		t.Fatal("deleted campaign retained")
	}
	if w := call("POST", "/api/admin/campaigns/AdminCamp00002/action", `{"action":"pause"}`, "42"); w.Code != 404 {
		t.Fatal("missing campaign action not 404")
	}
	if w := call("POST", "/api/admin/campaigns/AdminCamp00003/action", `{"action":"unknown"}`, "42"); w.Code != 400 {
		t.Fatal("invalid action accepted")
	}
}
