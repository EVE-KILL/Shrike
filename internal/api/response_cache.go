package api

import (
	"container/list"
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
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
}

type responseCacheBackend interface {
	Load(context.Context, string) (cachedResponse, time.Duration, bool)
	Store(context.Context, string, cachedResponse, time.Duration)
	DeleteMatching(context.Context, string)
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
	return &ResponseCache{
		local:  newResponseLRU(maxBytes),
		shared: shared,
	}
}

func (c *ResponseCache) Load(
	ctx context.Context,
	key string,
) (cachedResponse, bool) {
	if c == nil {
		return cachedResponse{}, false
	}
	if entry, ok := c.local.Get(key, time.Now()); ok {
		return entry, true
	}
	if c.shared == nil {
		return cachedResponse{}, false
	}
	entry, ttl, ok := c.shared.Load(ctx, key)
	if !ok {
		return cachedResponse{}, false
	}
	c.local.Put(key, entry, ttl, time.Now())
	return cloneCachedResponse(entry), true
}

func (c *ResponseCache) Store(
	ctx context.Context,
	key string,
	entry cachedResponse,
	ttl time.Duration,
) {
	if c == nil || ttl <= 0 {
		return
	}
	if entry.ContentType == "" {
		entry.ContentType = "application/json"
	}
	c.local.Put(key, entry, ttl, time.Now())
	if c.shared != nil {
		c.shared.Store(ctx, key, entry, ttl)
	}
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
	values := pipe.HMGet(ctx, key, "content_type", "body")
	remaining := pipe.PTTL(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return cachedResponse{}, 0, false
	}

	fields := values.Val()
	if len(fields) != 2 {
		return cachedResponse{}, 0, false
	}
	body, ok := fields[1].(string)
	if !ok {
		return cachedResponse{}, 0, false
	}
	ttl := remaining.Val()
	if ttl <= 0 {
		return cachedResponse{}, 0, false
	}
	contentType, _ := fields[0].(string)
	if contentType == "" {
		contentType = "application/json"
	}
	return cachedResponse{
		ContentType: contentType,
		Body:        []byte(body),
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
	pipe.HSet(ctx, key, "content_type", contentType, "body", entry.Body)
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
	}
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
