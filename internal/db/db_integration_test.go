package db

import (
	"os"
	"testing"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPrimaryAndReadPools(t *testing.T) {
	primaryURL := os.Getenv("TEST_DATABASE_URL")
	readURL := os.Getenv("TEST_DATABASE_READ_URL")
	if primaryURL == "" || readURL == "" {
		t.Skip("TEST_DATABASE_URL and TEST_DATABASE_READ_URL are required")
	}

	cfg := &config.Config{
		DatabaseURL: primaryURL, DatabaseMaxConnections: 2,
		DatabaseReadURL: readURL, DatabaseReadMaxConnections: 2,
	}
	primary, err := New(t.Context(), cfg)
	if err != nil {
		t.Fatalf("New primary: %v", err)
	}
	defer primary.Close()
	read, err := NewRead(t.Context(), cfg)
	if err != nil {
		t.Fatalf("New read: %v", err)
	}
	defer read.Close()

	assertApplicationName(t, primary, "shrike-primary")
	assertApplicationName(t, read, "shrike-read")

	if _, err := primary.Exec(t.Context(), "CREATE TEMP TABLE shrike_pool_write_test (id int)"); err != nil {
		t.Fatalf("primary write: %v", err)
	}
	if _, err := read.Exec(t.Context(), "CREATE TEMP TABLE shrike_pool_read_test (id int)"); err == nil {
		t.Fatal("read-only pool unexpectedly accepted a write")
	}
	var tableCount int
	if err := read.QueryRow(t.Context(),
		"SELECT count(*) FROM pg_tables WHERE schemaname = 'public'").Scan(&tableCount); err != nil {
		t.Fatalf("read pool query: %v", err)
	}
	if tableCount == 0 {
		t.Fatal("read pool found no migrated public tables")
	}
}

func assertApplicationName(t *testing.T, pool *pgxpool.Pool, want string) {
	t.Helper()
	var got string
	if err := pool.QueryRow(t.Context(), "SELECT current_setting('application_name')").Scan(&got); err != nil {
		t.Fatalf("application_name: %v", err)
	}
	if got != want {
		t.Fatalf("application_name = %q, want %q", got, want)
	}
}
