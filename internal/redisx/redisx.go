// Package redisx opens the two Valkey instances the services share.
//
// They are separate deliberately. Coordination state — rate-limit buckets,
// singleflight claims, the global pause flag — is small, hot, and must not be
// evicted; the response cache is large and entirely disposable. Putting them on
// one instance means a cache eviction policy that can throw away a rate-limit
// bucket, which is how a cluster ends up bursting ESI.
package redisx

import (
	"github.com/eve-kill/shrike/internal/config"
	"github.com/redis/go-redis/v9"
)

// Coordination opens the queue/coordination instance (REDIS_*).
func Coordination(cfg *config.Config) *redis.Client {
	if cfg.ValkeyQueueURL != "" {
		if options, err := redis.ParseURL(cfg.ValkeyQueueURL); err == nil {
			return redis.NewClient(options)
		}
	}
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
}

// Cache opens the response-cache instance (REDIS_CACHE_*, falling back to the
// coordination instance when unset, exactly as getCacheRedis() does).
//
// Password and DB are shared with the coordination instance; the TypeScript
// does not override them either, and diverging would silently point the two
// clients at different databases.
func Cache(cfg *config.Config) *redis.Client {
	if cfg.ValkeyCacheURL != "" {
		if options, err := redis.ParseURL(cfg.ValkeyCacheURL); err == nil {
			return redis.NewClient(options)
		}
	}
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisCacheAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
}
