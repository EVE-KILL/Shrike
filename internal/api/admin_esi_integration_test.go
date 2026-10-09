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

func TestAdminESIFiltersPollingEntitiesAndStoredDiscoveredKillmails(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_admin_esi_")
	if err := testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture owner','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture reader','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
		`INSERT INTO corporations(corporation_id,name,ticker) VALUES(10,'Fixture Red','FRED'),(11,'Fixture Blue','FBLUE')`,
		`INSERT INTO characters(character_id,name,corporation_id) VALUES(42,'Fixture owner',10),(43,'Fixture reader',11)`,
		`INSERT INTO esi_request_logs(character_id,endpoint,source,success,status_code,error_message,items_returned,new_items,new_item_ids,request_duration_ms) VALUES
		(42,'/characters/42/killmails/recent/','fixture-live',true,200,NULL,10,2,'[2000000001,2199999999]',4500),
		(42,'/corporations/10/killmails/recent/','fixture-live',false,304,NULL,0,0,'[]',50),
		(43,'/corporations/11/killmails/recent/','fixture-errors',false,503,'Upstream failure',0,0,'[]',5100),
		(43,'/characters/43/assets/','fixture-errors',false,400,'Permission denied',0,0,'[]',500)`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
	call := func(path, identity string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://localhost"+path, nil)
		if identity != "" {
			r.AddCookie(&http.Cookie{Name: "evelogin", Value: "00000000-0000-4000-8000-0000000000" + identity})
		}
		w := httptest.NewRecorder()
		service.Site().ServeHTTP(w, r)
		return w
	}
	for _, path := range []string{"/api/admin/esi", "/api/admin/esi-logs", "/api/admin/esi-entities?q=Fixture"} {
		for _, identity := range []string{"", "43"} {
			want := 401
			if identity == "43" {
				want = 403
			}
			w := call(path, identity)
			if w.Code != want || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatal("ESI private access failed")
			}
		}
	}
	type row struct {
		ID         int64
		Status     int     `json:"status_code"`
		Type       string  `json:"endpoint_type"`
		EntityName string  `json:"endpoint_entity_name"`
		Stored     []int64 `json:"stored_item_ids"`
		New        []int64 `json:"new_item_ids"`
	}
	type page struct {
		Rows         []row
		Total, Pages int
		Sources      []string
		NewRows      bool
	}
	read := func(query string) page {
		t.Helper()
		w := call("/api/admin/esi-logs?"+query, "42")
		var d page
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &d) != nil {
			t.Fatal("ESI read failed")
		}
		return d
	}
	all := read("limit=2")
	if all.Total != 4 || all.Pages != 2 || len(all.Rows) != 2 || len(all.Sources) != 2 {
		t.Fatal("ESI count/pages/source options diverged")
	}
	second := read("page=2&limit=2")
	if second.Rows[1].Stored[0] != 2000000001 || len(second.Rows[1].Stored) != 1 || len(second.Rows[1].New) != 2 || second.Rows[1].EntityName != "Fixture owner" {
		t.Fatal("stored/pending killmail projection diverged")
	}
	for _, check := range []struct {
		Query string
		Count int
	}{{"status=success", 2}, {"status=error", 2}, {"source=fixture-errors", 2}, {"character_id=42", 2}, {"corporation_id=11", 2}, {"endpoint_type=corporation", 2}, {"has_new=1", 1}, {"search=assets", 1}, {"search=missing", 0}, {"source=fixture-live&status=success&endpoint_type=character&has_new=1", 1}} {
		if got := read(check.Query); got.Total != check.Count {
			t.Errorf("%s=%d want %d", check.Query, got.Total, check.Count)
		}
	}
	top := all.Rows[0].ID
	if _, err := pool.Exec(ctx, `INSERT INTO esi_request_logs(character_id,endpoint,source,success,status_code,new_items,new_item_ids) VALUES(42,'/characters/42/killmails/recent/','fixture-live',true,200,1,'[2000000001]'),(43,'/characters/43/assets/','fixture-errors',false,503,0,'[]')`); err != nil {
		t.Fatal(err)
	}
	fresh := read("after_id=" + strconv.FormatInt(top, 10) + "&character_id=42&has_new=1")
	if !fresh.NewRows || len(fresh.Rows) != 1 || fresh.Rows[0].ID <= top {
		t.Fatal("live polling ignored cursor/filters")
	}
	entities := call("/api/admin/esi-entities?q=Fixture", "42")
	var found struct{ Results []map[string]any }
	if json.Unmarshal(entities.Body.Bytes(), &found) != nil || len(found.Results) != 4 {
		t.Fatal("entity search did not return logged users and corporations")
	}
}
