package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAdminAnnouncementsPersistSchedulesEditingArchiveAndAuthorization(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_admin_announcements_")
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture owner','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture reader','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}, responseCache: newResponseCache(newFakeResponseCacheBackend(), 4096)})
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
	publicRead := func() AnnouncementsResponse {
		w := call("GET", "/api/announcements", "", "")
		var d AnnouncementsResponse
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &d) != nil {
			t.Fatal("public announcement read failed")
		}
		return d
	}
	if len(publicRead().Announcements) != 0 {
		t.Fatal("initial public feed was not empty")
	}
	if w := call("GET", "/api/announcements", "", ""); w.Header().Get("X-Cache") != "HIT" {
		t.Fatal("public feed was not cached before mutations")
	}
	for _, identity := range []string{"", "43"} {
		want := 401
		if identity == "43" {
			want = 403
		}
		for _, method := range []string{"GET", "POST"} {
			if w := call(method, "/api/admin/announcements", `{}`, identity); w.Code != want {
				t.Fatal("unauthorized announcement access")
			}
		}
	}
	var ids []int64
	now := time.Now().UTC()
	for _, tier := range []int{1, 2, 3} {
		start, end := now.Add(-time.Minute), now.Add(time.Hour)
		if tier == 2 {
			start, end = now.Add(time.Hour), now.Add(2*time.Hour)
		}
		if tier == 3 {
			start, end = now.Add(-2*time.Hour), now.Add(-time.Hour)
		}
		body, _ := json.Marshal(map[string]any{"tier": tier, "title": " Fixture announcement ", "body_md": "**Fixture** <script>unsafe</script>", "color": "warning", "starts_at": start, "expires_at": end, "link_url": "/faq", "link_label": "Help"})
		w := call("POST", "/api/admin/announcements", string(body), "42")
		var d struct {
			Announcement struct {
				ID      int64
				Title   string
				HTML    string `json:"body_html"`
				Creator int64  `json:"created_by"`
			}
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &d) != nil || d.Announcement.ID == 0 || d.Announcement.Title != "Fixture announcement" || d.Announcement.Creator != 42 || strings.Contains(d.Announcement.HTML, "<script>") {
			t.Fatalf("announcement create/render failed (%d)", w.Code)
		}
		ids = append(ids, d.Announcement.ID)
	}
	if d := publicRead(); len(d.Announcements) != 1 || d.Announcements[0].ID != ids[0] {
		t.Fatal("creation did not invalidate cached public feed")
	}
	for _, status := range []string{"active", "scheduled", "expired"} {
		w := call("GET", "/api/admin/announcements?status="+status, "", "42")
		var d struct{ Announcements []map[string]any }
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &d) != nil || len(d.Announcements) != 1 || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
			t.Fatalf("announcement %s filtering failed", status)
		}
	}
	path := "/api/admin/announcements/" + strconv.FormatInt(ids[0], 10)
	if w := call("PATCH", path, `{"title":"Updated","icon":null,"link_url":null}`, "42"); w.Code != 200 {
		t.Fatal("announcement update failed")
	}
	if d := publicRead(); len(d.Announcements) != 1 || d.Announcements[0].Title != "Updated" {
		t.Fatal("edit retained stale public announcement")
	}
	if w := call("PUT", "/api/me/announcements/"+strconv.FormatInt(ids[0], 10)+"/dismissal", "", "43"); w.Code != 200 {
		t.Fatal("reader dismissal failed")
	}
	for _, identity := range []string{"42", "43"} {
		w := call("GET", "/api/me/announcements/dismissed", "", identity)
		var d DismissedAnnouncementIDsResponse
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &d) != nil || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("private dismissals failed")
		}
		want := 0
		if identity == "43" {
			want = 1
		}
		if len(d.DismissedIDs) != want {
			t.Fatal("dismissals leaked across accounts")
		}
	}
	if len(publicRead().Announcements) != 1 {
		t.Fatal("account dismissal changed shared public feed")
	}
	if w := call("PATCH", path, `{"expires_at":"2000-01-01T00:00:00Z"}`, "42"); w.Code != 400 {
		t.Fatal("invalid schedule accepted")
	}
	var title string
	var link *string
	var expiry time.Time
	if err := pool.QueryRow(ctx, `SELECT title,link_url,expires_at FROM announcements WHERE id=$1`, ids[0]).Scan(&title, &link, &expiry); err != nil || title != "Updated" || link != nil || expiry.Before(now) {
		t.Fatal("invalid edit damaged stored announcement")
	}
	if w := call("POST", path+"/archive", "", "43"); w.Code != 403 {
		t.Fatal("reader archive accepted")
	}
	for i := 0; i < 2; i++ {
		if w := call("POST", path+"/archive", "", "42"); w.Code != 200 {
			t.Fatal("archive/idempotency failed")
		}
	}
	if len(publicRead().Announcements) != 0 {
		t.Fatal("archive retained stale public announcement")
	}
	w := call("GET", "/api/admin/announcements?status=archived", "", "42")
	var d struct {
		Announcements []struct {
			ID       int64
			Archived *time.Time `json:"archived_at"`
		}
	}
	if json.Unmarshal(w.Body.Bytes(), &d) != nil || len(d.Announcements) != 1 || d.Announcements[0].ID != ids[0] || d.Announcements[0].Archived == nil {
		t.Fatal("archived announcement missing")
	}
}
