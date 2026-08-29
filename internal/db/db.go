// Package db owns the Postgres connection pool.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool tuning notes, which matter because every deployed service connects
// through pgbouncer rather than straight to Postgres:
//
//   - MaxConns stays deliberately low. The bouncer is the real connection cap,
//     and a large pgx pool behind a pooler just queues inside the bouncer while
//     consuming its slots — two poolers competing for the same budget.
//   - MinConns keeps connections warm. A cold connect through the bouncer costs
//     ~8 round trips (TCP + startup + SCRAM's three-message exchange +
//     ReadyForQuery), measured at 136-216ms over Tailscale against a 27ms warm
//     query. Without warm minimums, the first request on every new connection
//     pays that.
const (
	defaultMaxConns = 10
	defaultMinConns = 2
	connectTimeout  = 10 * time.Second
)

// New opens a pool and verifies it can actually serve a query. It returns only
// after a successful round trip, so callers never receive a pool that merely
// looks valid.
func New(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	maxConns := cfg.DatabaseMaxConnections
	if maxConns <= 0 {
		maxConns = defaultMaxConns
	}
	poolCfg.MaxConns = int32(maxConns)
	poolCfg.MinConns = defaultMinConns
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute

	// Statement caching is left at pgx's default (QueryExecModeCacheStatement).
	// That is safe against the production bouncer, which was verified to run in
	// SESSION pooling mode — a session-level SET survives to the next statement.
	// If pool_mode is ever changed to transaction, this must become
	// QueryExecModeExec or every prepared statement will start failing.

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return pool, nil
}
