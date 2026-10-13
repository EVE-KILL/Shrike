package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminUserDirectoryDetailAndMutationsRequireAdmin(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_admin_pages_")
	for _, query := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture owner','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture reader','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,last_login,created_at) SELECT 44000+i,'Fixture user '||to_char(i,'FM00'),'fixture',gen_random_uuid(),now()-i*interval '1 minute',now()-i*interval '1 day' FROM generate_series(1,54)i`,
		`INSERT INTO user_config(character_id,key,value) VALUES(43,'fixture','{"example":true}')`,
		`INSERT INTO user_esi_tokens(character_id,access_token,refresh_token,token_expiry,scopes) VALUES(42,'fixture-access','fixture-refresh',now()+interval '1 hour',ARRAY['esi-killmails.read_killmails.v1'])`,
	} {
		if _, err := pool.Exec(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
	call := func(method, path, body, identity string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		r.Header.Set("Origin", "http://localhost")
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
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
		for _, path := range []string{"/api/admin/overview", "/api/admin/users", "/api/admin/users/42", "/api/admin/esi"} {
			w := call("GET", path, "", identity)
			if w.Code != want || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatalf("private read %s=%d", path, w.Code)
			}
		}
		for _, path := range []string{"/api/admin/users/43/toggle-admin", "/api/admin/users/43/set-discord"} {
			if w := call("POST", path, `{"discord_user_id":"123456789012345"}`, identity); w.Code != want {
				t.Fatal("non-admin mutation accepted")
			}
		}
	}
	var page struct {
		Users []struct {
			ID   int64  `json:"character_id"`
			Name string `json:"character_name"`
		}
		Total, Pages int
	}
	first := call("GET", "/api/admin/users?sort=character_name&dir=asc&limit=50", "", "42")
	if err := json.Unmarshal(first.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if first.Code != 200 || len(page.Users) != 50 || page.Total != 56 || page.Pages != 2 {
		t.Fatalf("directory=%d total=%d pages=%d", len(page.Users), page.Total, page.Pages)
	}
	seen := map[int64]bool{}
	for _, user := range page.Users {
		seen[user.ID] = true
	}
	second := call("GET", "/api/admin/users?sort=character_name&dir=asc&limit=50&page=2", "", "42")
	if err := json.Unmarshal(second.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	for _, user := range page.Users {
		if seen[user.ID] {
			t.Fatal("admin page repeated a user")
		}
		seen[user.ID] = true
	}
	if len(seen) != 56 {
		t.Fatal("admin pagination omitted users")
	}
	search := call("GET", "/api/admin/users?search=44054", "", "42")
	if err := json.Unmarshal(search.Body.Bytes(), &page); err != nil || len(page.Users) != 1 || page.Users[0].ID != 44054 {
		t.Fatal("ID search diverged")
	}
	if w := call("GET", "/api/admin/users?search=missing", "", "42"); !strings.Contains(w.Body.String(), `"total":0`) {
		t.Fatal("empty admin search failed")
	}
	if w := call("GET", "/api/admin/users/999", "", "42"); w.Code != 404 {
		t.Fatal("missing admin user was not 404")
	}
	if w := call("POST", "/api/admin/users/42/toggle-admin", `{}`, "42"); w.Code != 400 {
		t.Fatal("admin could revoke their own access")
	}
	if w := call("POST", "/api/admin/users/43/set-discord", `{"discord_user_id":"123"}`, "42"); w.Code != 400 {
		t.Fatal("invalid Discord ID accepted")
	}
	for _, body := range []string{`{"discord_user_id":"123456789012345"}`, `{"discord_user_id":null}`} {
		if w := call("POST", "/api/admin/users/43/set-discord", body, "42"); w.Code != 200 {
			t.Fatal("Discord link update failed")
		}
		var id *string
		if err := pool.QueryRow(ctx, `SELECT discord_user_id FROM users WHERE character_id=43`).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if (id == nil) != (strings.Contains(body, "null")) {
			t.Fatal("Discord link not persisted")
		}
	}
	if w := call("POST", "/api/admin/users/43/toggle-admin", `{}`, "42"); w.Code != 200 {
		t.Fatal("fake account admin toggle failed")
	}
	if w := call("GET", "/api/admin/overview", "", "43"); w.Code != 200 {
		t.Fatal("granted role did not authorize reads")
	}
	if w := call("POST", "/api/admin/users/43/toggle-admin", `{}`, "42"); w.Code != 200 {
		t.Fatal("fake account admin removal failed")
	}
	if w := call("GET", "/api/admin/overview", "", "43"); w.Code != 403 {
		t.Fatal("revoked role still authorized reads")
	}
	detail := call("GET", "/api/admin/users/42", "", "42")
	if detail.Code != 200 || strings.Contains(detail.Body.String(), "fixture-access") || strings.Contains(detail.Body.String(), "fixture-refresh") || !strings.Contains(detail.Body.String(), "esi-killmails.read_killmails.v1") {
		t.Fatal("ESI token projection exposed credentials or omitted scopes")
	}
}
