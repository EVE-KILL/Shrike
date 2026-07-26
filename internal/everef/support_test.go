package everef

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The importers are I/O end to end: fetch a compressed archive, walk it, merge
// what comes out into Postgres. Almost nothing here is worth testing in
// isolation, and almost everything is worth testing against real bytes and a
// real database.
//
// The bytes come from testdata/ — a genuine bzip2'd CSV and a genuine tar.bz2,
// built once by hand because Go's standard library can decompress bzip2 but not
// produce it. Both are shaped like the real thing, including the parts that
// misbehave: another region's rows, a zero average, empty columns, a non-JSON
// file, and a truncated JSON file.
//
// The database is the local stack. Tests skip when it is unreachable.

const everefTestDSN = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"

// Reserved id bands, above anything CCP has issued, so nothing a test writes can
// collide with imported data.
const (
	testSystemBase = 39_000_000
	testWarBase    = 700_000
	testTypeBase   = 2_000_000
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = everefTestDSN
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// fileServer serves testdata files at the paths the importers construct, so the
// URL building is exercised rather than bypassed.
func fileServer(t *testing.T, routes map[string]string) *Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Errorf("read fixture %s: %v", name, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	return &Client{BaseURL: srv.URL, UserAgent: "shrike-test/1.0", HTTP: srv.Client()}
}

// jsonServer serves inline JSON bodies at exact paths.
func jsonServer(t *testing.T, routes map[string]string) *Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return &Client{BaseURL: srv.URL, UserAgent: "shrike-test/1.0", HTTP: srv.Client()}
}
