package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCommentsFeedSSRAndFragments(t *testing.T) {
	requests := []*http.Request{}
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Clone(r.Context()))
		if r.URL.Path != "/api/comments" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(commentFeedData{Comments: []feedComment{{
			ID: 26, TargetType: 1, TargetID: 138623545, CharacterID: 123, CharacterName: "Pilot",
			CorporationID: 456, CorporationName: "Corp", CreatedAt: time.Date(2026, 9, 22, 18, 49, 46, 0, time.UTC),
			BodyHTML: `<p>Hello</p><script>alert(1)</script>`, ReplyCount: 2,
		}}, NextCursor: func() *int64 { v := int64(22); return &v }()})
	})
	handler := New(Options{CommentsAPI: apiHandler})
	request := httptest.NewRequest("GET", "https://eve-kill.com/comments", nil)
	request.Header.Set("Cookie", "ek_session=test")
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, request)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Killmail #138623545") || !strings.Contains(page.Body.String(), "Hello") || strings.Contains(page.Body.String(), "<script>alert") {
		t.Fatalf("comments page response %d: %s", page.Code, page.Body.String())
	}
	if !strings.Contains(page.Body.String(), `data-controller="comments-feed"`) {
		t.Fatal("comments feed lost its component scope")
	}
	if len(requests) != 1 || requests[0].Host != "eve-kill.com" || requests[0].Header.Get("Cookie") != "ek_session=test" || requests[0].URL.Query().Get("limit") != "30" {
		t.Fatalf("API request did not retain site scope: %+v", requests)
	}
	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest("GET", "https://eve-kill.com/_fragments/comments?target_type=1&q=hello&cursor=55", nil))
	if fragment.Code != 200 || fragment.Header().Get("X-Next-Cursor") != "22" || !strings.Contains(fragment.Body.String(), "comment-feed-card") || !strings.Contains(fragment.Body.String(), "data-comments-fragment") {
		t.Fatalf("comment fragment %d: %s", fragment.Code, fragment.Body.String())
	}
	if len(requests) != 2 || requests[1].URL.Query().Get("target_type") != "1" || requests[1].URL.Query().Get("cursor") != "55" || requests[1].URL.Query().Get("q") != "hello" {
		t.Fatalf("comment fragment filters: %s", requests[1].URL.String())
	}
	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest("GET", "/_fragments/comments?target_type=9", nil))
	if invalid.Code != 400 {
		t.Fatalf("invalid filter status=%d", invalid.Code)
	}
}
