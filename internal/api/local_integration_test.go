package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Every test owns a newly created loopback database; cleanup never targets
// the supplied database or another test's state.
func localAPITestPool(t *testing.T, prefix string) (context.Context, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if !regexp.MustCompile(`^shrike_[a-z_]+_$`).MatchString(prefix) {
		t.Fatal("invalid local test prefix")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid local test database configuration")
	}
	if cfg.ConnConfig.Host != "127.0.0.1" && cfg.ConnConfig.Host != "localhost" {
		t.Fatal("integration tests require an explicit local PostgreSQL server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	var random [6]byte
	if _, err = rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	name := prefix + hex.EncodeToString(random[:])
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	var pool *pgxpool.Pool
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		clean, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := admin.Exec(clean, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("cleanup owned local integration database: %v", err)
		}
	})
	cfg = cfg.Copy()
	cfg.ConnConfig.Database = name
	pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = migrate.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return ctx, pool
}
