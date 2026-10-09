package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRemoteAdvancedSearchPreservesReadOnlyQueryContract(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.Header.Get("Cookie") != "" {
			t.Error("fixture reader transmitted a mutation or session")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/labels":
			w.Write([]byte(`{"labels":[{"id":"solo","name":"Solo"}]}`))
		case "/api/killlist/advanced":
			if r.URL.Query().Get("filters") != `{"iskMin":500000000}` || r.URL.Query().Get("after") != "123" || r.URL.Query().Get("view") != "fits" {
				t.Error("lost advanced query controls")
			}
			w.Write([]byte(`{"kills":[],"fits":[],"cursor":456,"hasMore":true}`))
		default:
			t.Errorf("unexpected read %s", r.URL.Path)
		}
	}))
	defer server.Close()
	reader := remoteAdvancedSearch{server.URL}
	labels, err := reader.Labels(context.Background())
	if err != nil || len(labels) != 1 || labels[0].ID != "solo" {
		t.Fatalf("labels=%v error=%v", labels, err)
	}
	result, err := reader.AdvancedSearch(context.Background(), url.Values{"filters": {`{"iskMin":500000000}`}, "after": {"123"}, "view": {"fits"}})
	if err != nil || !result.HasMore || result.Cursor == nil || *result.Cursor != 456 || calls != 2 {
		t.Fatalf("result=%v error=%v calls=%d", result, err, calls)
	}
}
