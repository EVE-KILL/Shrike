package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

func TestRemoteWarsUsesCredentialFreePublicReads(t *testing.T) {
	requests := make(chan string, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected public transport: %s", r.Method)
		}
		requests <- r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	reader := remoteWars{server.URL}
	ctx := context.Background()
	query := url.Values{"side": {"aggressor"}, "limit": {"100"}}
	if _, err := reader.WarDirectory(ctx, url.Values{"ongoing": {"true"}}, "private-board.example"); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.WarOverview(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.FactionWars(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := reader.WarDetail(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.WarKilllist(ctx, 42, query); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.WarStats(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.WarMembers(ctx, 42, query); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.WarIntel(ctx, 42); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 8)
	for i := range paths {
		paths[i] = <-requests
	}
	want := []string{"/api/conflicts/wars?ongoing=true", "/api/wars/stats", "/api/faction-wars", "/api/war/42", "/api/war/42/killlist?limit=100&side=aggressor", "/api/war/42/stats", "/api/war/42/members?limit=100&side=aggressor", "/api/war/42/intel"}
	if len(paths) != len(want) {
		t.Fatalf("paths: %v", paths)
	}
	for i, path := range want {
		if paths[i] != path {
			t.Errorf("path %d: got %s want %s", i, paths[i], path)
		}
	}
}
func TestRemoteWarNotFoundAndTransportFailures(t *testing.T) {
	var status atomic.Int32
	status.Store(http.StatusNotFound)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(int(status.Load())) }))
	defer server.Close()
	reader := remoteWars{server.URL}
	if _, found, err := reader.WarDetail(context.Background(), 42); found || err != nil {
		t.Fatalf("not found: found=%v err=%v", found, err)
	}
	status.Store(http.StatusServiceUnavailable)
	if _, found, err := reader.WarDetail(context.Background(), 42); found || err == nil {
		t.Fatalf("failure: found=%v err=%v", found, err)
	}
}
