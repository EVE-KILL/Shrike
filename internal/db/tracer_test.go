package db

import (
	"errors"
	"os"
	"testing"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/requeststats"
	"github.com/jackc/pgx/v5"
)

func TestTracingIsInactiveWithoutRequest(t *testing.T) {
	tracer := queryTracer{}
	ctx := t.Context()
	traced := tracer.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT secret", Args: []any{"private"}})
	if traced != ctx {
		t.Fatal("background tracing allocated a context")
	}
	tracer.TraceQueryEnd(traced, nil, pgx.TraceQueryEndData{Err: errors.New("query failure")})
}

func TestRequestTracingAgainstPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	cfg := &config.Config{DatabaseURL: databaseURL, DatabaseMaxConnections: 4, NodeEnv: "development"}
	pool, err := New(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx, stats := requeststats.Start(t.Context())
	var n int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&n); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(ctx, "SELECT generate_series(1,3)")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	batch := &pgx.Batch{}
	batch.Queue("SELECT 2")
	batch.Queue("SELECT 3")
	result := pool.SendBatch(ctx, batch)
	if _, err := result.Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := result.Exec(); err != nil {
		t.Fatal(err)
	}
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "SELECT 1/0"); err == nil {
		t.Fatal("expected query error")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "CREATE TEMP TABLE requeststats_test (id int)"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Conn().CopyFrom(ctx, pgx.Identifier{"requeststats_test"}, []string{"id"}, pgx.CopyFromRows([][]any{{1}, {2}})); err != nil {
		t.Fatal(err)
	}
	snapshot := stats.Snapshot()
	if snapshot.Queries != 7 || snapshot.Batches != 1 || snapshot.Copies != 1 || snapshot.Errors != 1 {
		t.Fatalf("query accounting = %+v", snapshot)
	}
	if snapshot.Database <= 0 || snapshot.PoolWait <= 0 {
		t.Fatalf("timings = %+v", snapshot)
	}
	cfg.NodeEnv = "production"
	production, err := New(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	if production.Config().ConnConfig.Tracer != nil {
		t.Fatal("production pool has development tracing")
	}
}
