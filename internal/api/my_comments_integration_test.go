package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMyCommentsCursorTargetsIsolationAndDeletion(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_my_comments_")
	const session = "00000000-0000-4000-8000-000000000042"
	for _, sql := range []string{
		`INSERT INTO users(character_id,character_name,character_owner_hash,session_id) VALUES(42,'Fixture Owner','fixture','00000000-0000-4000-8000-000000000042')`,
		`INSERT INTO user_sessions(session_id,character_id,expires_at) VALUES('00000000-0000-4000-8000-000000000042',42,now()+interval '1 hour')`,
		`INSERT INTO blog_posts(id,slug,title,status,author_id,author_name) VALUES(91,'fixture-article','Fixture article',1,42,'Fixture Owner')`,
		`INSERT INTO comments(id,target_type,target_id,body_md,body_html,character_id,character_name,corporation_id,corporation_name) VALUES(1,2,42,'First','<p>First</p>',42,'Fixture Owner',0,''),(2,9,91,'Article comment','<p>Article comment</p><script>alert(1)</script>',42,'Fixture Owner',0,''),(3,2,43,'Other account','<p>Other account</p>',43,'Fixture Other',0,'')`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}})
	call := func(method, path string, authed bool) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, "http://localhost"+path, nil)
		request.Header.Set("Origin", "http://localhost")
		if authed {
			request.AddCookie(&http.Cookie{Name: "evelogin", Value: session})
		}
		response := httptest.NewRecorder()
		service.Site().ServeHTTP(response, request)
		return response
	}
	type result struct {
		Comments []struct {
			ID   int64   `json:"id"`
			Slug *string `json:"target_slug"`
			HTML string  `json:"body_html"`
		}
		Cursor *int64 `json:"nextCursor"`
	}
	first := call("GET", "/api/user/comments?limit=1", true)
	var page result
	if err := json.Unmarshal(first.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if first.Code != 200 || len(page.Comments) != 1 || page.Comments[0].ID != 2 || page.Comments[0].Slug == nil || *page.Comments[0].Slug != "fixture-article" || strings.Contains(page.Comments[0].HTML, "<script") || page.Cursor == nil || *page.Cursor != 2 || first.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("private first page or derived article target: %d %+v", first.Code, page)
	}
	second := call("GET", "/api/user/comments?limit=1&cursor=2", true)
	page = result{}
	_ = json.Unmarshal(second.Body.Bytes(), &page)
	if second.Code != 200 || len(page.Comments) != 1 || page.Comments[0].ID != 1 || page.Cursor != nil {
		t.Fatalf("cursor or account isolation: %d %+v", second.Code, page)
	}
	if denied := call("DELETE", "/api/user/comments/3", true); denied.Code != 403 {
		t.Fatalf("other account delete=%d", denied.Code)
	}
	if deleted := call("DELETE", "/api/user/comments/2", true); deleted.Code != 200 {
		t.Fatalf("own delete=%d: %s", deleted.Code, deleted.Body.String())
	}
	last := call("GET", "/api/user/comments", true)
	page = result{}
	_ = json.Unmarshal(last.Body.Bytes(), &page)
	if last.Code != 200 || len(page.Comments) != 1 || page.Comments[0].ID != 1 {
		t.Fatalf("deleted row returned: %d %+v", last.Code, page)
	}
	if _, err := pool.Exec(ctx, `UPDATE user_sessions SET expires_at=now()-interval '1 minute' WHERE character_id=42`); err != nil {
		t.Fatal(err)
	}
	if expired := call("GET", "/api/user/comments", true); expired.Code != 401 {
		t.Fatalf("expired private list=%d", expired.Code)
	}
}
