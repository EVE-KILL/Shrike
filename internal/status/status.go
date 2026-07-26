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
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/zkb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// How often each tier is refreshed. The tick is one second; everything else is
// served from the last value gathered.
const (
	QueueInterval            = 5 * time.Second
	SlowInterval             = 60 * time.Second
	CoverageInterval         = 15 * time.Minute
	KillmailCoverageInterval = 10 * time.Minute

	eveKillCorporationID = 98779905
	isoMillisLayout      = "2006-01-02T15:04:05.000Z"
)

// Status is the published payload.
//
// Optional sources are pointers left nil rather than omitted, so the shape
// stays stable and a consumer can tell "not collected" from "collected as
// zero".
type Status struct {
	Event     string `json:"event"`
	Uptime    int64  `json:"uptime"`
	StartedAt string `json:"started_at"`
	Timestamp string `json:"timestamp"`

	System    SystemInfo           `json:"system"`
	Database  *DatabaseInfo        `json:"database"`
	Queues    map[string]QueueInfo `json:"queues"`
	ESI       *ESIInfo             `json:"esi"`
	Redis     *RedisInfo           `json:"redis"`
	Cache     *RedisInfo           `json:"redis_cache"`
	ZkbIngest *zkb.IngestStats     `json:"zkb_ingest"`
	Coverage  *CoverageInfo        `json:"coverage"`
	ESITokens *ESITokenInfo        `json:"esi_tokens"`
	Wallet    *WalletInfo          `json:"wallet"`

	// External HTTP-derived panels are intentionally deferred. Explicitly null
	// rather than absent, so the status page renders them as unknown.
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
	Size   string                       `json:"size"`
	Tables map[string]DatabaseTableInfo `json:"tables"`
}

type DatabaseTableInfo struct {
	TotalSize string `json:"total_size"`
	DataSize  string `json:"data_size"`
	Rows      int64  `json:"rows"`
}

// QueueInfo maps River states to the BullMQ-shaped contract the frontend
// consumes.
type QueueInfo struct {
	Waiting int64 `json:"waiting"`
	Active  int64 `json:"active"`
	Delayed int64 `json:"delayed"`
	Failed  int64 `json:"failed"`
}

// ESIInfo is the shared rate-limit and error budget.
type ESIInfo struct {
	TQStatus    string                  `json:"tq_status"`
	TQPlayers   *int64                  `json:"tq_players"`
	ErrorBudget int64                   `json:"error_budget"`
	Paused      bool                    `json:"paused"`
	Groups      map[string]ESIGroupInfo `json:"groups"`
}

type ESIGroupInfo struct {
	Remaining int64 `json:"remaining"`
	Limit     int64 `json:"limit"`
}

// RedisInfo is one instance's INFO.
type RedisInfo struct {
	Version                string `json:"version"`
	UptimeSeconds          int64  `json:"uptime_seconds"`
	ConnectedClients       int64  `json:"connected_clients"`
	UsedMemory             string `json:"used_memory"`
	TotalCommandsProcessed int64  `json:"total_commands_processed"`
	KeyspaceHits           int64  `json:"keyspace_hits"`
	KeyspaceMisses         int64  `json:"keyspace_misses"`
}

type WalletInfo struct {
	TotalBalance string  `json:"total_balance"`
	UpdatedAt    *string `json:"updated_at"`
}

type ESITokenInfo struct {
	Tokens  ESITokenSummary `json:"tokens"`
	Fetches ESIFetchSummary `json:"fetches"`
}

type ESITokenSummary struct {
	Total               int64 `json:"total"`
	Active              int64 `json:"active"`
	Revoked             int64 `json:"revoked"`
	CanFetchCharacter   int64 `json:"can_fetch_character"`
	CanFetchCorporation int64 `json:"can_fetch_corporation"`
}

type ESIFetchSummary struct {
	AllTime ESIFetchPeriod `json:"all_time"`
	Last24h ESIFetchPeriod `json:"last_24h"`
}

type ESIFetchPeriod struct {
	Total          int64 `json:"total"`
	KillmailsFound int64 `json:"killmails_found"`
	NewKillmails   int64 `json:"new_killmails"`
	Failed         int64 `json:"failed"`
}

type CoverageInfo struct {
	WindowDays   int               `json:"window_days"`
	Characters   CoverageEntity    `json:"characters"`
	Corporations CoverageEntity    `json:"corporations"`
	Alliances    CoverageAlliances `json:"alliances"`
}

type CoverageEntity struct {
	Active  int64 `json:"active"`
	Covered int64 `json:"covered"`
}

type CoverageAlliances struct {
	Active       int64 `json:"active"`
	CorpsActive  int64 `json:"corps_active"`
	CorpsCovered int64 `json:"corps_covered"`
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
	queues       map[string]QueueInfo
	lastQueues   time.Time
	database     *DatabaseInfo
	tokens       *ESITokenInfo
	wallet       *WalletInfo
	lastSlow     time.Time
	coverage     *CoverageInfo
	lastCoverage time.Time

	killmailsCovered     int64
	lastKillmailCoverage time.Time
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
		StartedAt: c.startedAt.Format(isoMillisLayout),
		Timestamp: now.Format(isoMillisLayout),
		System:    systemInfo(),
	}

	s.ESI = c.esiInfo(ctx)
	s.Redis = redisInfo(ctx, c.Redis)
	s.Cache = redisInfo(ctx, c.Cache)
	if stats, err := zkb.ReadIngestStats(ctx, c.Redis); err == nil {
		s.ZkbIngest = stats
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if now.Sub(c.lastQueues) >= QueueInterval {
		if queues := c.queueInfo(ctx); queues != nil {
			c.queues = queues
		}
		c.lastQueues = now
	}
	if now.Sub(c.lastKillmailCoverage) >= KillmailCoverageInterval {
		if n, ok := c.killmailsCovered24h(ctx); ok {
			c.killmailsCovered = n
		}
		c.lastKillmailCoverage = now
	}
	if now.Sub(c.lastSlow) >= SlowInterval {
		if db := c.databaseInfo(ctx); db != nil {
			c.database = db
		}
		if tokens := c.tokenInfo(ctx, c.killmailsCovered); tokens != nil {
			c.tokens = tokens
		}
		if wallet := c.walletInfo(ctx); wallet != nil {
			c.wallet = wallet
		}
		c.lastSlow = now
	}
	if now.Sub(c.lastCoverage) >= CoverageInterval {
		if cov := c.coverageInfo(ctx); cov != nil {
			c.coverage = cov
		}
		c.lastCoverage = now
	}
	s.Queues = c.queues
	s.Database = c.database
	s.ESITokens = c.tokens
	s.Wallet = c.wallet
	s.Coverage = c.coverage

	return s
}

func systemInfo() SystemInfo {
	total, free := hostMemory()
	var m runtime.MemStats
	if total == 0 {
		runtime.ReadMemStats(&m)
		total = m.Sys
		free = m.Sys - m.HeapInuse
	}

	return SystemInfo{
		Platform:    runtime.GOOS,
		Arch:        runtime.GOARCH,
		CPUs:        runtime.NumCPU(),
		MemoryTotal: formatBytes(total),
		MemoryFree:  formatBytes(free),
		LoadAverage: hostLoadAverage(),
	}
}

func hostMemory() (total, free uint64) {
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			continue
		}
		kib, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "MemTotal":
			total = kib * 1024
		case "MemAvailable":
			free = kib * 1024
		case "MemFree":
			if free == 0 {
				free = kib * 1024
			}
		}
	}
	return total, free
}

func hostLoadAverage() []float64 {
	raw, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return []float64{}
	}
	fields := strings.Fields(string(raw))
	out := make([]float64, 0, 3)
	for i := 0; i < len(fields) && i < 3; i++ {
		value, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return []float64{}
		}
		out = append(out, value)
	}
	return out
}

func (c *Collector) queueInfo(ctx context.Context) map[string]QueueInfo {
	depths, err := queue.StatusDepths(ctx, c.Pool)
	if err != nil {
		return nil
	}

	out := make(map[string]QueueInfo, len(depths))
	for _, d := range depths {
		out[d.Queue] = QueueInfo{
			Waiting: d.Available,
			Active:  d.Running,
			// Retryable jobs are waiting for their next scheduled attempt.
			Delayed: d.Scheduled + d.Retryable,
			Failed:  d.Discarded,
		}
	}
	return out
}

func (c *Collector) esiInfo(ctx context.Context) *ESIInfo {
	if c.Redis == nil {
		return nil
	}

	out := &ESIInfo{
		TQStatus: "unknown", ErrorBudget: 100,
		Groups: map[string]ESIGroupInfo{},
	}

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

	names := make([]string, 0, len(esi.Groups)+1)
	for name := range esi.Groups {
		names = append(names, name)
	}
	sort.Strings(names)
	names = append(names, esi.Probation.Name)

	keys := make([]string, len(names))
	for i, name := range names {
		keys[i] = "esi:tb:" + name + ":remaining"
	}
	values, err := c.Redis.MGet(ctx, keys...).Result()
	if err == nil {
		for i, name := range names {
			group := esi.Probation
			if configured, ok := esi.Groups[name]; ok {
				group = configured
			}
			remaining := int64(group.Limit)
			if values[i] != nil {
				if parsed, err := strconv.ParseInt(fmt.Sprint(values[i]), 10, 64); err == nil {
					remaining = parsed
				}
			}
			out.Groups[name] = ESIGroupInfo{
				Remaining: remaining,
				Limit:     int64(group.Limit),
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
		KeyspaceHits:           atoi(fields["keyspace_hits"]),
		KeyspaceMisses:         atoi(fields["keyspace_misses"]),
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
	out := &DatabaseInfo{Tables: map[string]DatabaseTableInfo{}}

	rows, err := c.Pool.Query(ctx, `
        SELECT c.relname,
               pg_size_pretty(pg_total_relation_size(c.oid)),
               pg_size_pretty(pg_relation_size(c.oid)),
               greatest(c.reltuples::bigint, s.n_live_tup)
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        LEFT JOIN pg_stat_user_tables s ON s.relid = c.oid
        WHERE n.nspname = 'public'
          AND c.relkind = 'r'
        ORDER BY pg_total_relation_size(c.oid) DESC`)
	if err != nil {
		return nil
	}
	for rows.Next() {
		var name, totalSize, dataSize string
		var rowCount int64
		if err := rows.Scan(&name, &totalSize, &dataSize, &rowCount); err != nil {
			rows.Close()
			return nil
		}
		out.Tables[name] = DatabaseTableInfo{
			TotalSize: totalSize,
			DataSize:  dataSize,
			Rows:      rowCount,
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
	out := &CoverageInfo{WindowDays: 30}
	if err := c.Pool.QueryRow(ctx, `
        WITH character_tokens AS (
            SELECT character_id
            FROM user_esi_tokens
            WHERE NOT disabled
              AND scopes @> ARRAY['esi-killmails.read_killmails.v1']::text[]
        ),
        corporation_tokens AS (
            SELECT DISTINCT c.corporation_id
            FROM user_esi_tokens token
            JOIN characters c ON c.character_id = token.character_id
            WHERE NOT token.disabled
              AND token.scopes @> ARRAY['esi-killmails.read_corporation_killmails.v1']::text[]
              AND c.corporation_id IS NOT NULL
        ),
        active AS (
            SELECT character.character_id, character.corporation_id, character.alliance_id,
                   (ct.character_id IS NOT NULL OR cpt.corporation_id IS NOT NULL) AS covered,
                   (cpt.corporation_id IS NOT NULL) AS corporation_covered
            FROM characters character
            LEFT JOIN character_tokens ct ON ct.character_id = character.character_id
            LEFT JOIN corporation_tokens cpt ON cpt.corporation_id = character.corporation_id
            WHERE character.last_active >= now() - interval '30 days'
        )
        SELECT count(*),
               count(*) FILTER (WHERE covered),
               count(DISTINCT corporation_id),
               count(DISTINCT corporation_id) FILTER (WHERE corporation_covered),
               count(DISTINCT alliance_id),
               count(DISTINCT corporation_id) FILTER (WHERE alliance_id IS NOT NULL),
               count(DISTINCT corporation_id) FILTER (
                   WHERE alliance_id IS NOT NULL AND corporation_covered
               )
        FROM active`).Scan(
		&out.Characters.Active,
		&out.Characters.Covered,
		&out.Corporations.Active,
		&out.Corporations.Covered,
		&out.Alliances.Active,
		&out.Alliances.CorpsActive,
		&out.Alliances.CorpsCovered,
	); err != nil {
		return nil
	}
	return out
}

func (c *Collector) walletInfo(ctx context.Context) *WalletInfo {
	out := &WalletInfo{}
	var updatedAt *time.Time
	if err := c.Pool.QueryRow(ctx, `
        SELECT coalesce(sum(balance), 0)::text, max(updated_at)
        FROM corporation_wallet_balances
        WHERE corporation_id = $1`, eveKillCorporationID).
		Scan(&out.TotalBalance, &updatedAt); err != nil {
		return nil
	}
	if updatedAt != nil {
		formatted := updatedAt.UTC().Format(isoMillisLayout)
		out.UpdatedAt = &formatted
	}
	return out
}

func (c *Collector) tokenInfo(ctx context.Context, killmailsCovered int64) *ESITokenInfo {
	out := &ESITokenInfo{}
	if err := c.Pool.QueryRow(ctx, `
        SELECT count(*),
               count(*) FILTER (WHERE scopes != '{}'),
               count(*) FILTER (
                   WHERE scopes @> ARRAY['esi-killmails.read_killmails.v1']::text[]
               ),
               count(*) FILTER (
                   WHERE scopes @> ARRAY['esi-killmails.read_corporation_killmails.v1']::text[]
               ),
               count(*) FILTER (WHERE scopes = '{}')
        FROM user_esi_tokens`).Scan(
		&out.Tokens.Total,
		&out.Tokens.Active,
		&out.Tokens.CanFetchCharacter,
		&out.Tokens.CanFetchCorporation,
		&out.Tokens.Revoked,
	); err != nil {
		return nil
	}

	if err := c.Pool.QueryRow(ctx, `
	        WITH per_character AS MATERIALIZED (
	            SELECT character_id,
	                   count(*)::bigint AS total,
	                   avg(items_returned) AS avg_items,
	                   coalesce(sum(new_items), 0)::bigint AS new_items,
	                   count(*) FILTER (WHERE NOT success)::bigint AS failed,
	                   count(*) FILTER (
	                       WHERE created_at > now() - interval '24 hours'
	                   )::bigint AS total_24h,
	                   coalesce(sum(new_items) FILTER (
	                       WHERE created_at > now() - interval '24 hours'
	                   ), 0)::bigint AS new_items_24h,
	                   count(*) FILTER (
	                       WHERE NOT success
	                         AND created_at > now() - interval '24 hours'
	                   )::bigint AS failed_24h
	            FROM esi_request_logs
	            WHERE source IN ('character_killmail_fetch', 'corp_killmail_fetch')
	            GROUP BY character_id
	        )
	        SELECT coalesce(sum(total), 0)::bigint,
	               coalesce(sum(avg_items), 0)::bigint,
	               coalesce(sum(new_items), 0)::bigint,
	               coalesce(sum(failed), 0)::bigint,
	               coalesce(sum(total_24h), 0)::bigint,
	               coalesce(sum(new_items_24h), 0)::bigint,
	               coalesce(sum(failed_24h), 0)::bigint
	        FROM per_character`).Scan(
		&out.Fetches.AllTime.Total,
		&out.Fetches.AllTime.KillmailsFound,
		&out.Fetches.AllTime.NewKillmails,
		&out.Fetches.AllTime.Failed,
		&out.Fetches.Last24h.Total,
		&out.Fetches.Last24h.NewKillmails,
		&out.Fetches.Last24h.Failed,
	); err != nil {
		return nil
	}
	out.Fetches.Last24h.KillmailsFound = killmailsCovered
	return out
}

func (c *Collector) killmailsCovered24h(ctx context.Context) (int64, bool) {
	var found int64
	err := c.Pool.QueryRow(ctx, `
        WITH character_tokens AS (
            SELECT character_id FROM user_esi_tokens
            WHERE NOT disabled
              AND scopes @> ARRAY['esi-killmails.read_killmails.v1']::text[]
        ),
        corporation_tokens AS (
            SELECT DISTINCT c.corporation_id
            FROM user_esi_tokens token
            JOIN characters c ON c.character_id = token.character_id
            WHERE NOT token.disabled
              AND token.scopes @> ARRAY['esi-killmails.read_corporation_killmails.v1']::text[]
              AND c.corporation_id IS NOT NULL
        )
        SELECT count(*) FROM (
            SELECT attacker.killmail_id
            FROM killmail_attackers attacker
            WHERE attacker.killmail_time >= now() - interval '24 hours'
              AND attacker.character_id IN (SELECT character_id FROM character_tokens)
            UNION
            SELECT attacker.killmail_id
            FROM killmail_attackers attacker
            WHERE attacker.killmail_time >= now() - interval '24 hours'
              AND attacker.corporation_id IN (
                  SELECT corporation_id FROM corporation_tokens
              )
            UNION
            SELECT killmail.killmail_id
            FROM killmails killmail
            WHERE killmail.killmail_time >= now() - interval '24 hours'
              AND killmail.victim_character_id IN (
                  SELECT character_id FROM character_tokens
              )
            UNION
            SELECT killmail.killmail_id
            FROM killmails killmail
            WHERE killmail.killmail_time >= now() - interval '24 hours'
              AND killmail.victim_corporation_id IN (
                  SELECT corporation_id FROM corporation_tokens
              )
        ) covered`).Scan(&found)
	return found, err == nil
}

func atoi(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func formatBytes(b uint64) string {
	const (
		megabyte = 1024 * 1024
		gigabyte = 1024 * megabyte
	)
	if b < gigabyte {
		return fmt.Sprintf("%.0f MB", float64(b)/megabyte)
	}
	return fmt.Sprintf("%.1f GB", float64(b)/gigabyte)
}
