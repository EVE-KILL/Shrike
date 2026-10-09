package api

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAccountOverviewQueryPreparesAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire test database connection: %v", err)
	}
	defer conn.Release()

	if _, err := conn.Conn().Prepare(ctx, "account_overview_regression", accountOverviewQuery); err != nil {
		t.Fatalf("prepare account overview query: %v", err)
	}
}

func TestAccountESIMetricsTimeWindowsAndConditionalRequests(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_account_esi_")
	if _, err := pool.Exec(ctx, `INSERT INTO users(character_id,character_name,character_owner_hash,session_id) VALUES(42,'Fixture ESI Owner','fixture','00000000-0000-4000-8000-000000000042'),(43,'Fixture ESI Other','fixture','00000000-0000-4000-8000-000000000043')`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 1, 20, 30, 0, 0, time.UTC)
	for _, entry := range []struct {
		id       int
		age      time.Duration
		status   int
		success  bool
		duration int
		items    int
	}{
		{42, 10 * time.Minute, 200, true, 100, 7},
		{42, 20 * time.Minute, 304, false, 200, 0},
		{42, 30 * time.Minute, 503, false, 300, 0},
		{42, 2 * time.Hour, 200, true, 400, 9},
		{42, 25 * time.Hour, 200, true, 500, 11},
		{43, 5 * time.Minute, 200, true, 9000, 1000},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO esi_request_logs(character_id,endpoint,source,created_at,status_code,success,request_duration_ms,new_items) VALUES($1,'/characters/42/killmails/recent/','local-regression',$2,$3,$4,$5,$6)`, entry.id, now.Add(-entry.age), entry.status, entry.success, entry.duration, entry.items); err != nil {
			t.Fatal(err)
		}
	}
	store := postgresAccountStore{db: pool}
	metrics, err := store.LoadESIMetrics(ctx, 42, now)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.RequestCount != 3 || metrics.AverageMS == nil || *metrics.AverageMS != 200 || metrics.P95MS == nil || *metrics.P95MS != 290 {
		t.Fatalf("hourly request metrics: %+v", metrics)
	}
	var total, errors, items int64
	for _, hour := range metrics.Volume {
		total += hour.Total
		errors += hour.Errors
		items += hour.NewItems
	}
	if total != 4 || errors != 1 || items != 16 {
		t.Fatalf("24-hour totals or account isolation differ: %d requests, %d errors, %d new items", total, errors, items)
	}
	empty, err := store.LoadESIMetrics(ctx, 44, now)
	if err != nil || empty.RequestCount != 0 || empty.AverageMS != nil || empty.P95MS != nil || len(empty.Volume) != 0 {
		t.Fatalf("empty account metrics: %+v, %v", empty, err)
	}
}
