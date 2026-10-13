package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRemoteLegacyPreservesArchiveFiltersAndAnonymousReads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("archive reader forwarded credentials or mutation")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/legacy/stats":
			_, _ = w.Write([]byte(`{"killmails":42}`))
		case "/api/legacy/top":
			if r.URL.Query().Get("dataType") != "characters" || r.URL.Query().Get("year") != "2005" || r.URL.Query().Get("limit") != "10" {
				t.Error("lost archive top controls")
			}
			_, _ = w.Write([]byte(`{"entries":[{"id":7,"name":"Historical pilot","count":3}]}`))
		case "/api/legacy/kills":
			if r.URL.Query().Get("system") != "Jita" || r.URL.Query().Get("after") != "123" || r.URL.Query().Get("sort") != "value_asc" {
				t.Error("lost archive filters/cursor/sort")
			}
			_, _ = w.Write([]byte(`{"kills":[],"cursor":99,"hasMore":true}`))
		case "/api/legacy/kill/7":
			_, _ = w.Write([]byte(`{"kill":{"killmail_id":7},"attackers":[],"items":[]}`))
		case "/api/legacy/kill/8":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Error("unexpected route", r.URL.Path)
		}
	}))
	defer server.Close()
	r, ctx := remoteLegacy{server.URL}, context.Background()
	if data, err := r.LegacyArchiveStats(ctx); err != nil || data.Killmails != 42 {
		t.Fatal(data, err)
	}
	if data, err := r.LegacyArchiveTop(ctx, "characters", 2005); err != nil || len(data) != 1 || data[0].ID != 7 {
		t.Fatal(data, err)
	}
	if data, err := r.LegacyArchiveKills(ctx, url.Values{"system": {"Jita"}, "after": {"123"}, "sort": {"value_asc"}}); err != nil || !data.HasMore || data.Cursor == nil || *data.Cursor != 99 {
		t.Fatal(data, err)
	}
	if data, found, err := r.LegacyArchiveKill(ctx, 7); err != nil || !found || data.Kill.KillmailID != 7 {
		t.Fatal(data, found, err)
	}
	if _, found, err := r.LegacyArchiveKill(ctx, 8); err != nil || found {
		t.Fatal(found, err)
	}
}

func TestLegacyReferenceRetainsArchiveMarkupAndAnonymousScope(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			t.Fatal("source received credentials")
		}
		_, _ = w.Write([]byte(`<div data-controller="legacy" data-legacy-page><div id="kill-list">Historical kill</div></div><script src="/assets/current/legacy.js"></script>`))
	})
	h := legacyReferenceRenderer(referenceRendererMustNotMutate{}, renderer, map[string][]byte{})
	for _, path := range []string{"/legacy", "/_fragments/legacy/kills", "/_fragments/legacy/top"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Cookie", "private")
		h.ServeHTTP(w, r)
		if !strings.Contains(w.Body.String(), `id="kill-list"`) || strings.Contains(w.Body.String(), `data-controller="legacy"`) || !strings.Contains(w.Body.String(), `/__fixture/legacy-source/legacy.js`) {
			t.Fatal(w.Body.String())
		}
	}
}
