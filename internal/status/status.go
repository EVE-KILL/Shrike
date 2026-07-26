// Package status assembles the system status the front page shows.
//
// It runs once a second, which is the constraint everything here is shaped by.
// The interesting numbers have wildly different costs — a Redis INFO is
// microseconds, a table-size query against a 600-million-row table is seconds —
// so they are gathered on separate schedules and cached between them. Fetching
// everything every tick would keep a database connection permanently busy
// answering a question nobody asked that second.
package status

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/zkb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// How often each tier is refreshed. The tick is one second; everything else is
// served from the last value gathered.
const (
	SlowInterval     = 60 * time.Second
	CoverageInterval = 5 * time.Minute
)

// Status is the published payload.
//
// Fields the Go port does not yet gather are pointers left nil rather than
// omitted, so the shape stays stable and a consumer can tell "not collected"
// from "collected as zero".
type Status struct {
	Event     string `json:"event"`
	Uptime    int64  `json:"uptime"`
	StartedAt string `json:"started_at"`
	Timestamp string `json:"timestamp"`

	System    SystemInfo       `json:"system"`
	Database  *DatabaseInfo    `json:"database"`
	Queues    *QueueInfo       `json:"queues"`
	ESI       *ESIInfo         `json:"esi"`
	Redis     *RedisInfo       `json:"redis"`
	Cache     *RedisInfo       `json:"redis_cache"`
	ZkbIngest *zkb.IngestStats `json:"zkb_ingest"`
	Coverage  *CoverageInfo    `json:"coverage"`

	// Not yet gathered by the Go port — the subsystems behind them are not
	// ported. Explicitly null rather than absent, so the status page renders
	// them as unknown instead of as broken.
	ESITokens   any `json:"esi_tokens"`
	Wallet      any `json:"wallet"`
	ImageServer any `json:"image_server"`
	WebSocket   any `json:"websocket"`
	Cloudflare  any `json:"cloudflare"`
}

// SystemInfo describes the host.
type SystemInfo struct {
	Platform    string    `json:"platform"`
	Arch        string    `json:"arch"`
	CPUs        int       `json:"cpus"`
	MemoryTotal string    `json:"memory_total"`
	MemoryFree  string    `json:"memory_free"`
	LoadAverage []float64 `json:"load_average"`
}

// DatabaseInfo is the expensive tier: table sizes and row estimates.
type DatabaseInfo struct {
	Killmails    int64  `json:"killmails"`
	Characters   int64  `json:"characters"`
	Corporations int64  `json:"corporations"`
	Alliances    int64  `json:"alliances"`
	Size         string `json:"size"`
}

// QueueInfo is the backlog per queue.
type QueueInfo struct {
	Queues map[string]int64 `json:"queues"`
	Total  int64            `json:"total"`
}

// ESIInfo is the shared rate-limit and error budget.
type ESIInfo struct {
	TQStatus    string           `json:"tq_status"`
	TQPlayers   *int64           `json:"tq_players"`
	ErrorBudget int64            `json:"error_budget"`
	Paused      bool             `json:"paused"`
	Groups      map[string]int64 `json:"groups"`
}

// RedisInfo is one instance's INFO.
type RedisInfo struct {
	Version                string `json:"version"`
	UptimeSeconds          int64  `json:"uptime_seconds"`
	ConnectedClients       int64  `json:"connected_clients"`
	UsedMemory             string `json:"used_memory"`
	TotalCommandsProcessed int64  `json:"total_commands_processed"`
}

// CoverageInfo is how complete the killmail record is.
type CoverageInfo struct {
	Killmails24h int64 `json:"killmails_24h"`
}

// Collector gathers status on tiered schedules.
//
// Holds the cached slow values between ticks, so it is stateful and one
// instance must be reused across runs rather than built per tick — a fresh one
// would refetch everything every second, which is exactly what the tiering
// exists to avoid.
type Collector struct {
	Pool  *pgxpool.Pool
	Redis *redis.Client
	Cache *redis.Client

	startedAt time.Time

	mu           sync.Mutex
	database     *DatabaseInfo
	lastSlow     time.Time
	coverage     *CoverageInfo
	lastCoverage time.Time
}

// NewCollector returns a collector that reports uptime from now.
func NewCollector(pool *pgxpool.Pool, coordination, cache *redis.Client) *Collector {
	return &Collector{
		Pool:      pool,
		Redis:     coordination,
		Cache:     cache,
		startedAt: time.Now().UTC(),
	}
}

// Collect assembles the current status.
//
// Every individual gatherer swallows its own failure and returns nil. A status
// page that shows nothing because one of eight sources is unwell is worse than
// one that shows seven sources and a gap — and this runs every second, so a
// transient failure resolves itself before anybody refreshes.
func (c *Collector) Collect(ctx context.Context) Status {
	now := time.Now().UTC()

	s := Status{
		Event:     "status",
		Uptime:    int64(now.Sub(c.startedAt).Seconds()),
		StartedAt: c.startedAt.Format(time.RFC3339),
		Timestamp: now.Format(time.RFC3339),
		System:    systemInfo(),
	}

	s.Queues = c.queueInfo(ctx)
	s.ESI = c.esiInfo(ctx)
	s.Redis = redisInfo(ctx, c.Redis)
	s.Cache = redisInfo(ctx, c.Cache)
	if stats, err := zkb.ReadIngestStats(ctx, c.Redis); err == nil {
		s.ZkbIngest = stats
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if now.Sub(c.lastSlow) >= SlowInterval {
		if db := c.databaseInfo(ctx); db != nil {
			c.database = db
		}
		c.lastSlow = now
	}
	if now.Sub(c.lastCoverage) >= CoverageInterval {
		if cov := c.coverageInfo(ctx); cov != nil {
			c.coverage = cov
		}
		c.lastCoverage = now
	}
	s.Database = c.database
	s.Coverage = c.coverage

	return s
}

func systemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		Platform: runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUs:     runtime.NumCPU(),
		// Go has no portable way to read host memory or load average without a
		// cgo dependency, so these report the process rather than the machine.
		// Named the same as the TypeScript's so the page renders, but they mean
		// something narrower.
		MemoryTotal: formatBytes(m.Sys),
		MemoryFree:  formatBytes(m.Sys - m.HeapInuse),
		LoadAverage: []float64{},
	}
}

func (c *Collector) queueInfo(ctx context.Context) *QueueInfo {
	depths, err := queue.Depths(ctx, c.Pool)
	if err != nil {
		return nil
	}

	out := &QueueInfo{Queues: map[string]int64{}}
	for _, d := range depths {
		pending := d.Pending()
		out.Queues[d.Queue] = pending
		out.Total += pending
	}
	return out
}

func (c *Collector) esiInfo(ctx context.Context) *ESIInfo {
	if c.Redis == nil {
		return nil
	}

	out := &ESIInfo{TQStatus: "unknown", ErrorBudget: 100, Groups: map[string]int64{}}

	if v, err := c.Redis.Get(ctx, "esi:tq:status").Result(); err == nil {
		out.TQStatus = v
	}
	if v, err := c.Redis.Get(ctx, "esi:tq:players").Int64(); err == nil {
		out.TQPlayers = &v
	}
	if v, err := c.Redis.Get(ctx, "esi:error:remaining").Int64(); err == nil {
		out.ErrorBudget = v
	}
	if err := c.Redis.Get(ctx, "esi:paused").Err(); err == nil {
		out.Paused = true
	}

	keys, err := c.Redis.Keys(ctx, "esi:group:*:remaining").Result()
	if err == nil {
		for _, k := range keys {
			name := strings.TrimSuffix(strings.TrimPrefix(k, "esi:group:"), ":remaining")
			if v, err := c.Redis.Get(ctx, k).Int64(); err == nil {
				out.Groups[name] = v
			}
		}
	}
	return out
}

func redisInfo(ctx context.Context, rdb *redis.Client) *RedisInfo {
	if rdb == nil {
		return nil
	}
	raw, err := rdb.Info(ctx).Result()
	if err != nil {
		return nil
	}

	fields := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, ":"); ok {
			fields[k] = v
		}
	}

	return &RedisInfo{
		Version:                fields["redis_version"],
		UptimeSeconds:          atoi(fields["uptime_in_seconds"]),
		ConnectedClients:       atoi(fields["connected_clients"]),
		UsedMemory:             fields["used_memory_human"],
		TotalCommandsProcessed: atoi(fields["total_commands_processed"]),
	}
}

// databaseInfo reads row counts from the planner statistics rather than with
// count(*).
//
// killmails has hundreds of millions of rows; counting them exactly takes
// minutes and locks nothing useful. reltuples is an estimate maintained by
// ANALYZE — which the analyze cron keeps current precisely so this is accurate
// enough to display.
func (c *Collector) databaseInfo(ctx context.Context) *DatabaseInfo {
	out := &DatabaseInfo{}

	rows, err := c.Pool.Query(ctx, `
        SELECT relname, greatest(reltuples, 0)::bigint
        FROM pg_class
        WHERE relname IN ('killmails','characters','corporations','alliances')
          AND relkind IN ('r','p')`)
	if err != nil {
		return nil
	}
	for rows.Next() {
		var name string
		var n int64
		if err := rows.Scan(&name, &n); err != nil {
			rows.Close()
			return nil
		}
		switch name {
		case "killmails":
			out.Killmails = n
		case "characters":
			out.Characters = n
		case "corporations":
			out.Corporations = n
		case "alliances":
			out.Alliances = n
		}
	}
	rows.Close()
	if rows.Err() != nil {
		return nil
	}

	if err := c.Pool.QueryRow(ctx,
		`SELECT pg_size_pretty(pg_database_size(current_database()))`).Scan(&out.Size); err != nil {
		return nil
	}
	return out
}

func (c *Collector) coverageInfo(ctx context.Context) *CoverageInfo {
	var n int64
	if err := c.Pool.QueryRow(ctx, `
        SELECT count(*) FROM killmails
        WHERE killmail_time > now() - interval '24 hours'`).Scan(&n); err != nil {
		return nil
	}
	return &CoverageInfo{Killmails24h: n}
}

func atoi(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTP"[exp])
}
