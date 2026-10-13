package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestMediaFixtureProviderPaginationAndErrors(t *testing.T) {
	f := newCommentMediaFixture()
	for _, page := range []string{"1", "2"} {
		request, _ := http.NewRequest(http.MethodGet, "https://api.klipy.com/api/v1/fixture/gifs/search?q=eve&page="+page, nil)
		response, err := f.RoundTrip(request)
		if err != nil {
			t.Fatal(err)
		}
		var data struct {
			Result bool
			Data   struct {
				Items   []any `json:"data"`
				HasNext bool  `json:"has_next"`
			}
		}
		if err = json.NewDecoder(response.Body).Decode(&data); err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if !data.Result || len(data.Data.Items) != 24 || data.Data.HasNext != (page == "1") {
			t.Fatalf("incorrect page %s: %+v", page, data)
		}
	}
	request, _ := http.NewRequest(http.MethodGet, "https://api.klipy.com/api/v1/fixture/gifs/search?q=provider-error", nil)
	response, err := f.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 503 {
		t.Fatalf("error fixture returned %d", response.StatusCode)
	}
}

func TestFacadeMediaProviderUsesLocalAssetsWithoutCallingTheRetainedAPI(t *testing.T) {
	calls := 0
	handler := newCommentMediaFixture().facade(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; http.Error(w, "unexpected provider read", 500) }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "http://account-review.localhost:4451/api/comments/klipy/trending?page=2", nil))
	var data struct {
		Items   []map[string]any `json:"items"`
		HasNext bool             `json:"has_next"`
		Page    int              `json:"page"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || len(data.Items) != 24 || data.HasNext || data.Page != 2 || calls != 0 {
		t.Fatalf("provider contract: status=%d rows=%d page=%d calls=%d", response.Code, len(data.Items), data.Page, calls)
	}
	if !strings.Contains(response.Body.String(), "http://account-review.localhost:4451/__fixture/media/") || strings.Contains(response.Body.String(), "localhost:4450/") {
		t.Fatal("facade assets escaped the local media handler")
	}
	failure := httptest.NewRecorder()
	handler.ServeHTTP(failure, httptest.NewRequest("GET", "http://account-review.localhost:4451/api/comments/klipy/search?q=provider-error", nil))
	if failure.Code != 503 || calls != 0 {
		t.Fatal("provider failure was not isolated")
	}
}

func TestFixturePublishesSuccessfulWritesInTheRealRelayFormat(t *testing.T) {
	f := newCommentMediaFixture()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("reject") == "yes" {
			http.Error(w, "rejected", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"comment":{"id":42,"target_type":10,"target_id":0,"target_slug":"PrizeWait00001","domain_id":null,"parent_id":null,"body_md":"Test"}}`)
	})
	server := httptest.NewServer(f.handler(next))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	connection, _, err := websocket.Dial(ctx, strings.Replace(server.URL, "http", "ws", 1)+"/ws/comments", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.CloseNow()
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/comments", strings.NewReader(`{}`))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	_, frame, err := connection.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var event struct {
		Channel string
		Data    struct {
			EventType  string `json:"event_type"`
			CommentID  int    `json:"comment_id"`
			TargetSlug string `json:"target_slug"`
		}
	}
	if err = json.Unmarshal(frame, &event); err != nil {
		t.Fatal(err)
	}
	if event.Channel != "comments" || event.Data.EventType != "new" || event.Data.CommentID != 42 || event.Data.TargetSlug != "PrizeWait00001" {
		t.Fatalf("bad relay frame: %s", frame)
	}
	if strings.Contains(string(frame), "body_md") {
		t.Fatalf("private Markdown was added to the fixture relay: %s", frame)
	}
}
