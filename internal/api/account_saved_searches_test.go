package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSavedSearchValidation(t *testing.T) {
	for _, raw := range []string{
		`{"version":2,"filters":{},"view":"kills","dedup":"family"}`,
		`{"version":1,"filters":{"timeRange":{"preset":"forever"}},"view":"kills","dedup":"family"}`,
		`{"version":1,"filters":{"sort":{"field":"random SQL","direction":"asc"}},"view":"kills","dedup":"family"}`,
		`{"version":1,"filters":{"iskMin":100,"iskMax":10},"view":"kills","dedup":"family"}`,
		`{"version":1,"filters":{"location":{"securityTypes":["bogus"]}},"view":"kills","dedup":"family"}`,
		`{"version":1,"filters":{"entities":{"victim":[{"id":-1,"type":"character"}]}},"view":"kills","dedup":"family"}`,
	} {
		var document savedSearchDocument
		if err := json.Unmarshal([]byte(raw), &document); err != nil {
			t.Fatal(err)
		}
		if err := validateSavedSearch(&savedSearchBody{Name: "test", Document: document}); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	body := savedSearchBody{Name: "  Wormholes  ", Document: savedSearchDocument{Version: 1, View: "kills", Dedup: "family"}}
	if err := validateSavedSearch(&body); err != nil {
		t.Fatal(err)
	}
	if body.Name != "Wormholes" || body.Document.Filters.TimeRange.Preset != "30d" {
		t.Fatalf("defaults: %#v", body)
	}
}

// Runs only on the explicitly configured disposable database. It creates an
// isolated schema, applies the actual migration, and exercises real HTTP/SQL.
func TestSavedSearchAccountIsolationAndConcurrency(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("saved_search_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `CREATE TABLE users (character_id integer PRIMARY KEY); INSERT INTO users VALUES (1),(2),(3)`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/00026_account_saved_searches.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, down, _ := strings.Cut(string(migration), "-- +goose Down")
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatal(err)
	}
	handler := func(owner int32) http.Handler {
		mux := http.NewServeMux()
		a := humago.New(mux, huma.DefaultConfig("saved search test", "1"))
		now := time.Now()
		store := &fakeAuthStore{now: now, principal: &Principal{CharacterID: owner}}
		registerSavedSearchRoutes(a, Options{DB: pool, Auth: AuthOptions{store: store, now: func() time.Time { return now }}}, nil)
		return mux
	}
	h1, h2, h3 := handler(1), handler(2), handler(3)
	call := func(h http.Handler, method, path, body string, authenticated bool, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "https://eve-kill.com"+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if authenticated {
			req.AddCookie(&http.Cookie{Name: authSessionCookie, Value: "b4529550-a8a5-43d8-8342-909719305ef0"})
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	create := func(name string) string {
		raw, _ := json.Marshal(savedSearchBody{Name: name, Document: savedSearchDocument{Version: 1, View: "kills", Dedup: "family"}})
		return string(raw)
	}
	assertCode := func(rec *httptest.ResponseRecorder, code int) {
		t.Helper()
		if rec.Code != code {
			t.Fatalf("status %d want %d: %s", rec.Code, code, rec.Body.String())
		}
	}
	assertCode(call(h1, "POST", "/me/saved-searches", create("Private"), false, ""), 401)
	assertCode(call(h1, "POST", "/me/saved-searches", create("Private"), true, "https://evil.example"), 403)
	created := call(h1, "POST", "/me/saved-searches", create("Private"), true, "")
	assertCode(created, 200)
	var record savedSearchRecord
	if err := json.Unmarshal(created.Body.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(created.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("private response can be cached")
	}
	assertCode(call(h1, "POST", "/me/saved-searches", create(" PRIVATE "), true, ""), 409)
	listed := call(h2, "GET", "/me/saved-searches", "", true, "")
	assertCode(listed, 200)
	if strings.Contains(listed.Body.String(), "Private") {
		t.Fatal("other account can read search")
	}
	path := fmt.Sprintf("/me/saved-searches/%d", record.ID)
	updated, _ := json.Marshal(savedSearchBody{Name: "Renamed", Document: record.Document, Revision: 1})
	assertCode(call(h2, "PUT", path, string(updated), true, ""), 409)
	assertCode(call(h2, "DELETE", path+"?revision=1", "", true, ""), 409)
	assertCode(call(h1, "PUT", path, string(updated), true, ""), 200)
	assertCode(call(h1, "PUT", path, string(updated), true, ""), 409)
	assertCode(call(h1, "DELETE", path+"?revision=1", "", true, ""), 409)
	assertCode(call(h1, "DELETE", path+"?revision=2", "", true, ""), 200)
	// Race past the limit: account locking must make the cap exact.
	var wg sync.WaitGroup
	codes := make(chan int, 60)
	for i := 0; i < 60; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes <- call(h3, "POST", "/me/saved-searches", create(fmt.Sprintf("Search %d", i)), true, "").Code
		}(i)
	}
	wg.Wait()
	close(codes)
	success, conflict := 0, 0
	for code := range codes {
		switch code {
		case 200:
			success++
		case 409:
			conflict++
		default:
			t.Errorf("unexpected concurrent status %d", code)
		}
	}
	if success != 50 || conflict != 10 {
		t.Fatalf("limit: %d created, %d rejected", success, conflict)
	}
	if _, err := pool.Exec(ctx, down); err != nil {
		t.Fatalf("rollback: %v", err)
	}
}
