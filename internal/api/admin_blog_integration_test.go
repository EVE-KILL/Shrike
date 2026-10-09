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

func TestAdminBlogCRUDPrivacyFiltersDraftPreviewAndConflicts(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_admin_blog_")
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture author','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture reader','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
	call := func(method, path, body, identity string) *httptest.ResponseRecorder {
		t.Helper()
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
		for _, action := range []struct{ method, path string }{{"GET", "/api/admin/blog"}, {"GET", "/api/admin/blog/1"}, {"GET", "/api/admin/blog/preview/draft"}, {"POST", "/api/admin/blog"}, {"PATCH", "/api/admin/blog/1"}, {"DELETE", "/api/admin/blog/1"}} {
			w := call(action.method, action.path, `{}`, identity)
			if w.Code != want || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatalf("blog privacy %s %s: %d", action.method, action.path, w.Code)
			}
		}
	}
	decode := func(w *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var d struct{ Post map[string]any }
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &d) != nil || d.Post == nil {
			t.Fatalf("blog operation failed: %d %s", w.Code, w.Body.String())
		}
		return d.Post
	}
	post := decode(call("POST", "/api/admin/blog", `{"title":"  Fixture draft  ","body_md":"## Heading\n\n**Safe** <script>unsafe</script>","tags":["PvP","pvp","UPDATES"],"excerpt":"  summary  ","cover_image_url":"/__fixture/blog-cover.svg"}`, "42"))
	id := int64(post["id"].(float64))
	path := "/api/admin/blog/" + strconv.FormatInt(id, 10)
	slug := post["slug"].(string)
	if post["title"] != "Fixture draft" || post["author_id"] != float64(42) || post["published_at"] != nil || strings.Contains(post["body_html"].(string), "<script>") {
		t.Fatal("draft create or Markdown sanitization failed")
	}
	if w := call("GET", "/api/blog/"+slug, "", ""); w.Code != 404 {
		t.Fatal("draft visible publicly")
	}
	if p := decode(call("GET", "/api/admin/blog/preview/"+slug, "", "42")); p["id"] != float64(id) {
		t.Fatal("private draft preview missing")
	}
	if w := call("POST", "/api/admin/blog", `{"title":"Other","slug":"fixture-draft","body_md":"different"}`, "42"); w.Code != 409 {
		t.Fatal("duplicate slug accepted")
	}
	if w := call("POST", "/api/admin/blog", `{"title":"   "}`, "42"); w.Code != 400 {
		t.Fatal("empty title accepted")
	}
	future := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	body, _ := json.Marshal(map[string]any{"title": "Updated <script>plain title</script>", "slug": "fixture-renamed", "body_md": "# Revised\n\n[Unsafe](javascript:alert(1))\n\n**Still safe**", "excerpt": nil, "cover_image_url": nil, "tags": []string{"NEWS", "news", "two words"}, "status": 1, "published_at": future})
	p := decode(call("PATCH", path, string(body), "42"))
	if p["slug"] != "fixture-renamed" || p["excerpt"] != nil || p["cover_image_url"] != nil || strings.Contains(p["body_html"].(string), `href="javascript:`) {
		t.Fatal("blog update lost nullable fields or safe rendered Markdown")
	}
	var storedDate time.Time
	var title string
	if err := pool.QueryRow(ctx, `SELECT title,published_at FROM blog_posts WHERE id=$1`, id).Scan(&title, &storedDate); err != nil || !storedDate.Equal(future) {
		t.Fatal("EVE publication date did not persist")
	}
	for _, invalid := range []string{`{"title":" "}`, `{"published_at":"bad"}`, `{"status":5}`, `{"slug":"!!!"}`} {
		if w := call("PATCH", path, invalid, "42"); w.Code != 400 {
			t.Fatal("invalid edit accepted")
		}
	}
	if p := decode(call("GET", path, "", "42")); p["title"] != title {
		t.Fatal("invalid update damaged existing post")
	}
	decode(call("PATCH", path, `{"status":2}`, "42"))
	decode(call("GET", "/api/blog/fixture-renamed", "", ""))
	decode(call("POST", "/api/admin/blog", `{"title":"Separate draft","slug":"separate-draft","body_md":"draft"}`, "42"))
	decode(call("POST", "/api/admin/blog", `{"title":"Live","slug":"live","body_md":"public","status":1}`, "42"))
	if _, err := pool.Exec(ctx, `UPDATE blog_posts SET created_at='2026-10-05T12:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	for _, filter := range []string{"draft", "published", "archived", "all"} {
		w := call("GET", "/api/admin/blog?status="+filter+"&limit=100", "", "42")
		var d struct {
			Posts []struct {
				ID     int64
				Status int
			}
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &d) != nil || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("private blog listing failed")
		}
		want := 1
		if filter == "all" {
			want = 3
		}
		if len(d.Posts) != want {
			t.Fatalf("%s filter size %d", filter, len(d.Posts))
		}
		for i, p := range d.Posts {
			if i > 0 && p.ID >= d.Posts[i-1].ID {
				t.Fatal("tied creation dates reordered blog list")
			}
		}
	}
	decode(call("PATCH", path, `{"status":0,"published_at":null}`, "42"))
	if w := call("GET", "/api/blog/fixture-renamed", "", ""); w.Code != 404 {
		t.Fatal("returning to draft left public post visible")
	}
	if w := call("DELETE", path, "", "42"); w.Code != 200 {
		t.Fatal("local blog deletion failed")
	}
	for _, action := range []string{"GET", "PATCH", "DELETE"} {
		if w := call(action, path, `{}`, "42"); w.Code != 404 {
			t.Fatal("deleted blog post remained writable")
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM blog_posts WHERE id=$1`, id).Scan(&count); err != nil || count != 0 {
		t.Fatal("deleted post remained in PostgreSQL")
	}
}
