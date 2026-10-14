// Package redisx opens the shared Valkey used for caches, coordination, and
// pub/sub. River stores its queues in Postgres and does not use Valkey.
package redisx

import (
	"context"
	"strconv"
	"sync/atomic"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/redis/go-redis/v9"
)

var poolID atomic.Uint64

// New opens the shared Valkey configured by REDIS_*.
func New(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if telemetry.Enabled() {
		client.AddHook(telemetryHook{})
		id := strconv.FormatUint(poolID.Add(1), 10)
		telemetry.RegisterCollector("redis.pool."+id, func(ctx context.Context) {
			stats := client.PoolStats()
			attrs := telemetry.Attributes("pool", id)
			telemetry.Gauge(ctx, "redis.pool.connections", float64(stats.TotalConns), attrs)
			telemetry.Gauge(ctx, "redis.pool.idle", float64(stats.IdleConns), attrs)
			telemetry.Gauge(ctx, "redis.pool.timeouts", float64(stats.Timeouts), attrs)
			telemetry.Gauge(ctx, "redis.pool.hits", float64(stats.Hits), attrs)
			telemetry.Gauge(ctx, "redis.pool.misses", float64(stats.Misses), attrs)
		})
	}
	return client
}
