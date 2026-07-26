package status

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func TestStatusJSONContract(t *testing.T) {
	payload := Status{
		Event:     "status",
		StartedAt: "2026-07-26T04:00:00.000Z",
		Timestamp: "2026-07-26T04:00:01.000Z",
		Database: &DatabaseInfo{
			Size: "1 GB",
			Tables: map[string]DatabaseTableInfo{
				"killmails": {TotalSize: "1 GB", DataSize: "900 MB", Rows: 42},
			},
		},
		Queues: map[string]QueueInfo{
			"killmails": {Waiting: 3, Active: 1, Delayed: 2, Failed: 0},
		},
		Coverage: &CoverageInfo{
			WindowDays: 30,
			Characters: CoverageEntity{Active: 10, Covered: 8},
		},
		ESITokens: &ESITokenInfo{
			Tokens: ESITokenSummary{Total: 5, Active: 4},
		},
		Wallet: &WalletInfo{TotalBalance: "123.45"},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}

	queues := decoded["queues"].(map[string]any)
	killmails := queues["killmails"].(map[string]any)
	if killmails["waiting"] != float64(3) || killmails["active"] != float64(1) {
		t.Fatalf("queue payload = %#v", killmails)
	}

	database := decoded["database"].(map[string]any)
	if _, ok := database["tables"].(map[string]any)["killmails"]; !ok {
		t.Fatalf("database payload = %#v", database)
	}
	if decoded["coverage"].(map[string]any)["window_days"] != float64(30) {
		t.Fatalf("coverage payload = %#v", decoded["coverage"])
	}
	if decoded["esi_tokens"].(map[string]any)["tokens"] == nil {
		t.Fatalf("token payload = %#v", decoded["esi_tokens"])
	}
	if decoded["wallet"].(map[string]any)["total_balance"] != "123.45" {
		t.Fatalf("wallet payload = %#v", decoded["wallet"])
	}
}

// Opt-in because the coverage queries intentionally scan production-shaped
// data. Run locally with STATUS_INTEGRATION_TEST=1 when changing the collector.
func TestCollectorAgainstServices(t *testing.T) {
	if os.Getenv("STATUS_INTEGRATION_TEST") != "1" {
		t.Skip("set STATUS_INTEGRATION_TEST=1")
	}

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr, DB: 14})
	defer rdb.Close() //nolint:errcheck
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}

	payload := NewCollector(pool, rdb, rdb).Collect(ctx)
	if payload.Database == nil || len(payload.Database.Tables) == 0 {
		t.Fatal("database stats were not collected")
	}
	if payload.Queues == nil {
		t.Fatal("queue stats were not collected")
	}
	if payload.ESITokens == nil {
		t.Fatal("ESI token stats were not collected")
	}
	if payload.Wallet == nil {
		t.Fatal("wallet stats were not collected")
	}
	if payload.Coverage == nil {
		t.Fatal("coverage stats were not collected")
	}
}
