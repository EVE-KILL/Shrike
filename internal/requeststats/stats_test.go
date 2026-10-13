package requeststats

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConcurrentRequestStatsStayIsolated(t *testing.T) {
	ctx, first := Start(t.Context())
	other, second := Start(t.Context())
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() { s := From(ctx); s.AddQuery(); s.AddDatabase(time.Millisecond, false) })
		wg.Go(func() { s := From(other); s.AddQuery(); s.AddDatabase(2*time.Millisecond, true) })
	}
	wg.Wait()
	a, b := first.Snapshot(), second.Snapshot()
	if a.Queries != 100 || b.Queries != 100 || a.Errors != 0 || b.Errors != 100 || a.Database != 100*time.Millisecond || b.Database != 200*time.Millisecond {
		t.Fatalf("snapshots: %+v %+v", a, b)
	}
	From(t.Context()).AddQuery() // Background calls have no collector.
	if first.Snapshot().Queries != 100 {
		t.Fatal("background work affected request")
	}
}

func TestAPITimingIsMeasuredBeforeHeadersAndPreservesFlusher(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Error("Flusher was lost")
		}
		stats := From(r.Context())
		stats.AddQuery()
		stats.AddDatabase(2*time.Millisecond, false)
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = w.Write([]byte("ok"))
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/search", nil))
	headers := w.Result().Header
	if !strings.Contains(headers.Get("Server-Timing"), `dbcount;desc="1"`) || !strings.Contains(headers.Get("Server-Timing"), "db;dur=2.000") {
		t.Fatalf("timing = %q", headers.Get("Server-Timing"))
	}
	if headers.Get("Cache-Control") != "private, no-store" {
		t.Fatal("development timings may be browser-cached")
	}
}

func TestStreamingFeedIsNotWrapped(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if From(r.Context()) != nil {
			t.Fatal("SSE has a request collector")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/feed/stream", nil))
	if w.Header().Get("Server-Timing") != "" {
		t.Fatal("SSE response profiled")
	}
}
