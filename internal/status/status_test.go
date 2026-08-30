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
			Size: "1 GB", Version: "18.6", Role: "primary",
			SampledAt: "2026-07-26T04:00:01.000Z", UptimeSeconds: 3600,
			Connections: DatabaseConnectionInfo{
				Total: 10, Active: 2, IdleInTransaction: 1, Waiting: 1, Max: 100,
			},
			Cluster: DatabaseClusterInfo{
				Replicas: 2, Streaming: 2, Synchronous: 1, MaxLagBytes: 1024, MaxLagSeconds: 0.25,
			},
			Workload: &DatabaseWorkloadInfo{
				TransactionsPerSecond: 42.5, RollbackPercent: 0.1,
				WALBytesPerSecond: 1024, ReadBytesPerSecond: 2048, WriteBytesPerSecond: 512,
				ReadLatencyMS: 0.2, WriteLatencyMS: 0.4, RowsChangedPerSecond: 50, TempBytesPerSecond: 128,
			},
			Statements:  &DatabaseStatementInfo{QueriesPerSecond: 125, AverageLatencyMS: 3.25},
			Maintenance: &DatabaseMaintenanceInfo{Checkpoints: 1, ActiveAutovacuums: 2, DeadRows: 1000},
			History: []DatabaseHistoryPoint{{
				Timestamp: "2026-07-26T04:00:01.000Z", QueriesPerSecond: 125, TransactionsPerSecond: 42.5,
			}},
			CacheHitRatio: 99.95,
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
	if database["role"] != "primary" || database["version"] != "18.6" {
		t.Fatalf("database identity payload = %#v", database)
	}
	connections := database["connections"].(map[string]any)
	if connections["active"] != float64(2) || connections["max"] != float64(100) {
		t.Fatalf("database connection payload = %#v", connections)
	}
	cluster := database["cluster"].(map[string]any)
	if cluster["streaming"] != float64(2) || cluster["max_lag_bytes"] != float64(1024) {
		t.Fatalf("database cluster payload = %#v", cluster)
	}
	statements := database["statements"].(map[string]any)
	if statements["queries_per_second"] != float64(125) || statements["average_latency_ms"] != float64(3.25) {
		t.Fatalf("database statement payload = %#v", statements)
	}
	if len(database["history"].([]any)) != 1 {
		t.Fatalf("database history payload = %#v", database["history"])
	}
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

func TestCounterDeltaHandlesStatisticsReset(t *testing.T) {
	if got := counterDelta(20, 15); got != 5 {
		t.Fatalf("counter delta = %d, want 5", got)
	}
	if got := counterDelta(2, 15); got != 0 {
		t.Fatalf("reset counter delta = %d, want 0", got)
	}
	if got := floatCounterDelta(2.5, 1.25); got != 1.25 {
		t.Fatalf("float counter delta = %f, want 1.25", got)
	}
	if got := floatCounterDelta(0.5, 1.25); got != 0 {
		t.Fatalf("reset float counter delta = %f, want 0", got)
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

	payload := NewCollector(pool, rdb).Collect(ctx)
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

func TestDatabaseMetricsAgainstPostgres(t *testing.T) {
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

	collector := NewCollector(pool, nil)
	first := collector.databaseInfo(ctx)
	if first == nil || collector.dbSnapshot == nil {
		t.Fatal("initial database metrics were not collected")
	}
	collector.dbSnapshot.at = collector.dbSnapshot.at.Add(-time.Second)
	if _, err := pool.Exec(ctx, `SELECT 1`); err != nil {
		t.Fatal(err)
	}
	second := collector.databaseInfo(ctx)
	if second == nil || second.Workload == nil {
		t.Fatal("database workload rates were not collected")
	}
	if second.Statements == nil {
		t.Fatal("pg_stat_statements rates were not collected")
	}
	if second.Maintenance == nil || len(second.History) != 1 {
		t.Fatalf("database maintenance/history metrics = %#v", second)
	}
	if second.SampledAt == "" || second.Role == "" {
		t.Fatalf("database identity metrics = %#v", second)
	}
}
