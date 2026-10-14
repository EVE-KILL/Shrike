package api

import (
	"container/list"
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// ResponseCache is the API response cache.
//
// L1 is a byte-bounded LRU in this process. L2 is the shared Valkey. A read
// checks L1, promotes an L2 hit into L1 for the L2 entry's remaining lifetime,
// and otherwise lets the handler load the source. A source response is written
// through to both tiers.
type ResponseCache struct {
	local  *responseLRU
	shared responseCacheBackend
	loads  singleflight.Group
}

type responseCacheBackend interface {
	Load(context.Context, string) (cachedResponse, time.Duration, bool)
	Store(context.Context, string, cachedResponse, time.Duration)
	DeleteMatching(context.Context, string)
}

type cachedResponseWithTTL struct {
	Entry cachedResponse
	TTL   time.Duration
}

type responseCacheBulkBackend interface {
	LoadMany(context.Context, []string) map[string]cachedResponseWithTTL
}

// LoadOnce collapses concurrent cache misses for the same key in this process.
// The shared Valkey cache handles reuse across replicas; this guard prevents a
// burst inside one replica from running the same expensive database query for
// every waiting request before the first response reaches Valkey.
func (c *ResponseCache) LoadOnce(
	ctx context.Context,
	key string,
	load func() (any, error),
) (any, error, bool) {
	if c == nil {
		value, err := load()
		return value, err, false
	}
	result := c.loads.DoChan(key, load)
	select {
	case <-ctx.Done():
		return nil, ctx.Err(), false
	case value := <-result:
		return value.Val, value.Err, value.Shared
	}
}

// Refresh starts one detached load for a key in this process. Callers use it
// after serving a stale response, so the client request never owns the refresh.
func (c *ResponseCache) Refresh(key string, load func() (any, error)) {
	c.RefreshContext(context.Background(), key, func(context.Context) (any, error) { return load() })
}

// RefreshContext starts a separate background transaction; an already-served
// HTTP request must not retain spans for a refresh that continues afterwards.
func (c *ResponseCache) RefreshContext(ctx context.Context, key string, load func(context.Context) (any, error)) {
	if c == nil {
		return
	}
	carrier := telemetry.Propagation(ctx)
	go func() {
		_, _, _ = c.loads.Do(key, func() (any, error) {
			ctx, task := telemetry.StartTaskWithTrace(context.Background(), "cache.refresh", "API cache refresh", carrier)
			defer telemetry.RecoverAndRepanic(ctx)
			value, err := load(ctx)
			telemetry.Capture(ctx, err)
			outcome := "completed"
			if err != nil {
				outcome = "failed"
			}
			task.End(outcome, err)
			return value, err
		})
	}()
}

// NewResponseCache builds an L1 cache over a shared Valkey L2.
//
// maxBytes is a hard payload ceiling for this process. Zero disables L1 but
// leaves L2 enabled. A nil client disables L2 but still permits a local cache,
// which is useful during a temporary Valkey outage and in focused tests.
func NewResponseCache(client *redis.Client, maxBytes int64) *ResponseCache {
	var shared responseCacheBackend
	if client != nil {
		shared = &redisResponseCache{client: client}
	}
	return newResponseCache(shared, maxBytes)
}

func newResponseCache(shared responseCacheBackend, maxBytes int64) *ResponseCache {
	cache := &ResponseCache{
		local:  newResponseLRU(maxBytes),
		shared: shared,
	}
	telemetry.RegisterCollector("cache.api", func(ctx context.Context) {
		cache.local.mu.Lock()
		entries, bytes, maximum := len(cache.local.items), cache.local.bytes, cache.local.maxBytes
		cache.local.mu.Unlock()
		attrs := telemetry.Attributes("cache", "api", "tier", "l1")
		telemetry.Gauge(ctx, "cache.entries", float64(entries), attrs)
		telemetry.Gauge(ctx, "cache.bytes", float64(bytes), attrs, telemetry.WithUnit(sentry.UnitByte))
		telemetry.Gauge(ctx, "cache.capacity", float64(maximum), attrs, telemetry.WithUnit(sentry.UnitByte))
	})
	return cache
}

func (c *ResponseCache) Load(
	ctx context.Context,
	key string,
) (cachedResponse, bool) {
	entry, fresh, ok := c.LoadState(ctx, key)
	return entry, ok && fresh
}

// LoadMany reads fresh entries in one shared-cache round trip after checking
// L1. It never loads the database or starts refresh work on a miss.
func (c *ResponseCache) LoadMany(ctx context.Context, keys []string) map[string]cachedResponse {
	result := make(map[string]cachedResponse)
	if c == nil {
		return result
	}
	now := time.Now()
	missing := []string{}
	seen := map[string]bool{}
	for _, key := range keys {
		if seen[key] {
			continue
		}
		seen[key] = true
		if entry, ok := c.local.Get(key, now); ok && !sharedOnlyResponse(key) {
			if responseIsFresh(entry, now) {
				result[key] = entry
			}
			continue
		}
		missing = append(missing, key)
	}
	if c.shared == nil || len(missing) == 0 {
		return result
	}
	if shared, ok := c.shared.(responseCacheBulkBackend); ok {
		for key, value := range shared.LoadMany(ctx, missing) {
			if !sharedOnlyResponse(key) {
				c.local.Put(key, value.Entry, value.TTL, time.Now())
			}
			if responseIsFresh(value.Entry, time.Now()) {
				result[key] = cloneCachedResponse(value.Entry)
			}
		}
	} else {
		for _, key := range missing {
			if entry, ok := c.Load(ctx, key); ok {
				result[key] = entry
			}
		}
	}
	return result
}

// LoadState returns fresh and stale entries while their retention TTL lasts.
func (c *ResponseCache) LoadState(
	ctx context.Context,
	key string,
) (cachedResponse, bool, bool) {
	if c == nil {
		return cachedResponse{}, false, false
	}
	ctx, span := telemetry.StartSpan(ctx, "cache.get", "API response cache")
	started := time.Now()
	tier, outcome := "l2", "miss"
	defer func() {
		attrs := telemetry.Attributes("cache", "api", "tier", tier, "outcome", outcome)
		telemetry.Count(ctx, "cache.lookup", 1, attrs)
		telemetry.Duration(ctx, "cache.lookup.duration", time.Since(started), attrs)
		if span != nil {
			span.SetData("cache.hit", outcome != "miss")
			span.SetData("cache.tier", tier)
			span.Status = sentry.SpanStatusOK
			span.Finish()
		}
	}()
	if entry, ok := c.local.Get(key, time.Now()); ok && !sharedOnlyResponse(key) {
		tier, outcome = "l1", "hit"
		if !responseIsFresh(entry, time.Now()) {
			outcome = "stale"
		}
		return entry, responseIsFresh(entry, time.Now()), true
	}
	if c.shared == nil {
		return cachedResponse{}, false, false
	}
	entry, ttl, ok := c.shared.Load(ctx, key)
	if !ok {
		return cachedResponse{}, false, false
	}
	outcome = "hit"
	if !responseIsFresh(entry, time.Now()) {
		outcome = "stale"
	}
	if !sharedOnlyResponse(key) {
		c.local.Put(key, entry, ttl, time.Now())
	}
	return cloneCachedResponse(entry), responseIsFresh(entry, time.Now()), true
}

func (c *ResponseCache) Store(
	ctx context.Context,
	key string,
	entry cachedResponse,
	ttl time.Duration,
) {
	c.store(ctx, key, entry, ttl, 0)
}

// StoreStale retains an entry after its fresh TTL so a request can serve it
// while a detached refresh replaces it.
func (c *ResponseCache) StoreStale(
	ctx context.Context,
	key string,
	entry cachedResponse,
	freshTTL time.Duration,
	staleTTL time.Duration,
) {
	c.store(ctx, key, entry, freshTTL, staleTTL)
}

func (c *ResponseCache) store(
	ctx context.Context,
	key string,
	entry cachedResponse,
	freshTTL time.Duration,
	staleTTL time.Duration,
) {
	if c == nil || freshTTL <= 0 {
		return
	}
	if entry.ContentType == "" {
		entry.ContentType = "application/json"
	}
	entry.FreshUntil = time.Now().Add(freshTTL)
	retentionTTL := freshTTL + staleTTL
	if !sharedOnlyResponse(key) {
		c.local.Put(key, entry, retentionTTL, time.Now())
	}
	if c.shared != nil {
		c.shared.Store(ctx, key, entry, retentionTTL)
	}
}

// User-editable conflict, coalition and announcement data is invalidated on mutation. Keep
// these entries only in shared storage: a process-local copy cannot observe a
// deletion performed by another replica (or recover missed pub/sub messages).
// If shared storage is unavailable, these routes safely fall back to their DB.
func sharedOnlyResponse(key string) bool {
	return strings.Contains(key, "battle") || strings.Contains(key, "/coalitions") || strings.Contains(key, "announcements:canonical") || strings.Contains(key, "/announcements/active")
}

// DeleteMatching invalidates both tiers. The pattern uses the Redis glob
// syntax needed by the current callers: '*' and '?' wildcards.
func (c *ResponseCache) DeleteMatching(ctx context.Context, pattern string) {
	if c == nil {
		return
	}
	if c.shared != nil {
		c.shared.DeleteMatching(ctx, pattern)
	}
	// Remove L1 last so a concurrent L2 promotion cannot leave a stale local
	// entry after this call completes.
	c.local.DeleteMatching(pattern)
}

type redisResponseCache struct {
	client *redis.Client
}

func (c *redisResponseCache) Load(
	ctx context.Context,
	key string,
) (cachedResponse, time.Duration, bool) {
	pipe := c.client.Pipeline()
	values := pipe.HMGet(ctx, key, "content_type", "body", "fresh_until")
	remaining := pipe.PTTL(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return cachedResponse{}, 0, false
	}

	return decodeRedisCachedResponse(values.Val(), remaining.Val())
}

func (c *redisResponseCache) LoadMany(ctx context.Context, keys []string) map[string]cachedResponseWithTTL {
	result := make(map[string]cachedResponseWithTTL)
	pipe := c.client.Pipeline()
	values := make([]*redis.SliceCmd, len(keys))
	remaining := make([]*redis.DurationCmd, len(keys))
	for i, key := range keys {
		values[i] = pipe.HMGet(ctx, key, "content_type", "body", "fresh_until")
		remaining[i] = pipe.PTTL(ctx, key)
	}
	if len(keys) == 0 {
		return result
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return result
	}
	for i, key := range keys {
		entry, ttl, ok := decodeRedisCachedResponse(values[i].Val(), remaining[i].Val())
		if ok {
			result[key] = cachedResponseWithTTL{Entry: entry, TTL: ttl}
		}
	}
	return result
}

func decodeRedisCachedResponse(fields []any, ttl time.Duration) (cachedResponse, time.Duration, bool) {
	if len(fields) != 3 {
		return cachedResponse{}, 0, false
	}
	body, ok := fields[1].(string)
	if !ok {
		return cachedResponse{}, 0, false
	}
	if ttl <= 0 {
		return cachedResponse{}, 0, false
	}
	contentType, _ := fields[0].(string)
	if contentType == "" {
		contentType = "application/json"
	}
	var freshUntil time.Time
	if raw, ok := fields[2].(string); ok {
		if unixMillis, err := strconv.ParseInt(raw, 10, 64); err == nil {
			freshUntil = time.UnixMilli(unixMillis)
		}
	}
	return cachedResponse{
		ContentType: contentType,
		Body:        []byte(body),
		FreshUntil:  freshUntil,
	}, ttl, true
}

func (c *redisResponseCache) Store(
	ctx context.Context,
	key string,
	entry cachedResponse,
	ttl time.Duration,
) {
	contentType := entry.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	pipe := c.client.TxPipeline()
	pipe.HSet(ctx, key,
		"content_type", contentType,
		"body", entry.Body,
		"fresh_until", entry.FreshUntil.UnixMilli(),
	)
	pipe.PExpire(ctx, key, ttl)
	// The handler already produced the correct response. A cache failure must
	// never turn that response into an application failure.
	_, _ = pipe.Exec(ctx)
}

func (c *redisResponseCache) DeleteMatching(
	ctx context.Context,
	pattern string,
) {
	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, pattern, 250).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return
			}
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

type responseLRU struct {
	mu       sync.Mutex
	maxBytes int64
	bytes    int64
	items    map[string]*list.Element
	order    list.List
}

type responseLRUEntry struct {
	key       string
	response  cachedResponse
	expiresAt time.Time
	bytes     int64
}

func newResponseLRU(maxBytes int64) *responseLRU {
	return &responseLRU{
		maxBytes: maxBytes,
		items:    make(map[string]*list.Element),
	}
}

func (c *responseLRU) Get(
	key string,
	now time.Time,
) (cachedResponse, bool) {
	if c == nil || c.maxBytes <= 0 {
		return cachedResponse{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return cachedResponse{}, false
	}
	entry := element.Value.(*responseLRUEntry)
	if !now.Before(entry.expiresAt) {
		c.remove(element)
		return cachedResponse{}, false
	}
	c.order.MoveToFront(element)
	return cloneCachedResponse(entry.response), true
}

func (c *responseLRU) Put(
	key string,
	response cachedResponse,
	ttl time.Duration,
	now time.Time,
) {
	if c == nil || c.maxBytes <= 0 || ttl <= 0 {
		return
	}
	cloned := cloneCachedResponse(response)
	size := cachedResponseSize(key, cloned)

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.items[key]; ok {
		c.remove(existing)
	}
	if size > c.maxBytes {
		return
	}
	entry := &responseLRUEntry{
		key:       key,
		response:  cloned,
		expiresAt: now.Add(ttl),
		bytes:     size,
	}
	element := c.order.PushFront(entry)
	c.items[key] = element
	c.bytes += size

	for c.bytes > c.maxBytes {
		c.remove(c.order.Back())
	}
}

func (c *responseLRU) DeleteMatching(pattern string) {
	if c == nil || c.maxBytes <= 0 {
		return
	}
	match := cacheGlob(pattern)

	c.mu.Lock()
	defer c.mu.Unlock()
	for key, element := range c.items {
		if match.MatchString(key) {
			c.remove(element)
		}
	}
}

func (c *responseLRU) remove(element *list.Element) {
	if element == nil {
		return
	}
	entry := element.Value.(*responseLRUEntry)
	delete(c.items, entry.key)
	c.order.Remove(element)
	c.bytes -= entry.bytes
}

func cloneCachedResponse(entry cachedResponse) cachedResponse {
	return cachedResponse{
		ContentType: entry.ContentType,
		Body:        append([]byte(nil), entry.Body...),
		FreshUntil:  entry.FreshUntil,
	}
}

func responseIsFresh(entry cachedResponse, now time.Time) bool {
	return entry.FreshUntil.IsZero() || now.Before(entry.FreshUntil)
}

func cachedResponseSize(key string, entry cachedResponse) int64 {
	return int64(len(key) + len(entry.ContentType) + len(entry.Body))
}

func cacheGlob(pattern string) *regexp.Regexp {
	quoted := regexp.QuoteMeta(pattern)
	quoted = strings.ReplaceAll(quoted, `\*`, `.*`)
	quoted = strings.ReplaceAll(quoted, `\?`, `.`)
	return regexp.MustCompile(`^` + quoted + `$`)
}
