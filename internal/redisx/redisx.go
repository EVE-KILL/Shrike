// Package redisx opens the shared Valkey used for caches, coordination, and
// pub/sub. River stores its queues in Postgres and does not use Valkey.
package redisx

import (
	"github.com/eve-kill/shrike/internal/config"
	"github.com/redis/go-redis/v9"
)

// New opens the shared Valkey configured by REDIS_*.
func New(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
}
