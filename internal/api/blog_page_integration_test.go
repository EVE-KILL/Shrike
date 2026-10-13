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

func TestBlogReaderAndAPIVisibilityAuthorMarkdownAndCompleteCursorPages(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_blog_pages_")
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id,is_admin) VALUES(42,'Fixture author','fixture','00000000-0000-4000-8000-000000000042',true),(43,'Fixture reader','fixture','00000000-0000-4000-8000-000000000043',false)`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) SELECT session_id,character_id,now()+interval '1 hour' FROM users`,
		`INSERT INTO alliances(alliance_id,name,ticker) VALUES(10,'Fixture alliance','FALL')`,
		`INSERT INTO corporations(corporation_id,name,ticker,alliance_id) VALUES(11,'Fixture corporation','FCOR',10)`,
		`INSERT INTO characters(character_id,name,corporation_id,alliance_id) VALUES(42,'Fixture author',11,10)`,
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
	base := time.Now().UTC().Truncate(time.Second).Add(123456 * time.Microsecond)
	for i := 1; i <= 25; i++ {
		offset := i
		if i == 19 || i == 21 {
			offset = 20
		}
		body, _ := json.Marshal(map[string]any{"slug": "published-" + strconv.Itoa(i), "title": "Published " + strconv.Itoa(i), "body_md": "## Heading\n\n**Bold** [link](https://eve-kill.com)\n\n<script>alert(1)</script>", "status": 1, "published_at": base.Add(-time.Duration(offset) * time.Hour), "tags": []string{"fixture", "updates"}})
		if w := call("POST", "/api/admin/blog", string(body), "42"); w.Code != 200 {
			t.Fatalf("create=%d", w.Code)
		}
	}
	for _, status := range []int{0, 2} {
		body, _ := json.Marshal(map[string]any{"slug": "state-" + strconv.Itoa(status), "title": "Private draft or archive", "body_md": "Body", "status": status})
		if w := call("POST", "/api/admin/blog", string(body), "42"); w.Code != 200 {
			t.Fatal("state creation failed")
		}
	}
	reader := service.Homepage()
	first, err := reader.PublicBlogPosts(ctx, nil, "updates", 20)
	if err != nil || len(first.Posts) != 20 || first.NextCursor == "" {
		t.Fatalf("reader first page: %v", err)
	}
	var apiPage struct {
		Posts []BlogListPost `json:"posts"`
		Next  string         `json:"nextCursor"`
	}
	response := call("GET", "/api/blog?limit=20&tag=updates", "", "")
	if err := json.Unmarshal(response.Body.Bytes(), &apiPage); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || apiPage.Next != first.NextCursor || len(apiPage.Posts) != len(first.Posts) {
		t.Fatal("API and primary reader pagination diverged")
	}
	cursor, err := ParseBlogCursor(first.NextCursor)
	if err != nil || cursor.ID != first.Posts[19].ID || !cursor.PublishedAt.Equal(*first.Posts[19].PublishedAt) {
		t.Fatal("cursor lost publication precision or ID")
	}
	second, err := reader.PublicBlogPosts(ctx, &cursor, "updates", 20)
	if err != nil || len(second.Posts) != 5 || second.NextCursor != "" {
		t.Fatalf("reader second page: %v", err)
	}
	seen := map[int64]bool{}
	for _, post := range apiPage.Posts {
		seen[post.ID] = true
	}
	response = call("GET", "/api/blog?limit=20&tag=updates&cursor="+first.NextCursor, "", "")
	apiPage.Next = "" // JSON null does not overwrite an existing Go string.
	if err := json.Unmarshal(response.Body.Bytes(), &apiPage); err != nil {
		t.Fatal(err)
	}
	for i, post := range apiPage.Posts {
		if i >= len(second.Posts) || post.ID != second.Posts[i].ID {
			t.Fatal("API and reader next-page ordering diverged")
		}
		if seen[post.ID] {
			t.Fatal("cursor repeated a post")
		}
		seen[post.ID] = true
	}
	if len(seen) != 25 || apiPage.Next != "" {
		t.Fatalf("cursor skipped tied publication times: %d/25 posts", len(seen))
	}
	if w := call("GET", "/api/blog?cursor=v1:invalid", "", ""); w.Code != 400 {
		t.Fatalf("invalid API cursor=%d", w.Code)
	}
	post, found, err := reader.PublicBlogPost(ctx, "published-1")
	if err != nil || !found || post.AuthorCorporationName == nil || *post.AuthorCorporationName != "Fixture corporation" || post.AuthorAllianceName == nil || strings.Contains(post.BodyHTML, "<script") || !strings.Contains(post.BodyHTML, "<strong>Bold</strong>") {
		t.Fatal("author snapshot or sanitized article diverged")
	}
	if _, found, err := reader.PublicBlogPost(ctx, "state-0"); err != nil || found {
		t.Fatal("public reader exposed draft")
	}
	if _, found, err := reader.PublicBlogPost(ctx, "state-2"); err != nil || !found {
		t.Fatal("archive disappeared from its public URL")
	}
	for _, account := range []string{"", "43", "42"} {
		w := call("GET", "/api/admin/blog/preview/state-0", "", account)
		expected := 401
		if account == "43" {
			expected = 403
		}
		if account == "42" {
			expected = 200
		}
		if w.Code != expected || !strings.Contains(w.Header().Get("Cache-Control"), "private, no-store") {
			t.Fatalf("draft access %q=%d", account, w.Code)
		}
	}
}
