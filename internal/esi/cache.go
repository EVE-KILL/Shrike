package esi

import (
	"container/list"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// The response cache, in two tiers.
//
// ESI tells us how long every response stays valid, and most of what a
// killboard asks for barely changes: a corporation's name, an alliance's ticker,
// a killmail that is immutable by construction. Honouring `Expires` is most of
// what keeps the request rate survivable.
//
// L1 is in-process and costs nothing to consult. L2 is shared, so one worker
// filling the cache serves every other and a restart does not start cold.
//
// Entries keep their ETag past expiry, because an expired entry is still worth
// something: a conditional request that comes back 304 refreshes it without
// spending error budget or transferring a body.

const (
	cacheKeyPrefix = "esi:v2:cache:"
	// A floor and ceiling on what an Expires header may claim. ESI occasionally
	// serves a date in the past, and a 0-second TTL would make the cache useless
	// exactly when it is needed most.
	minCacheTTL = 60 * time.Second
	maxCacheTTL = 24 * time.Hour
	// localCacheMax bounds the in-process tier. Entries are small and mostly
	// character blobs; this is a few tens of megabytes at worst.
	localCacheMax = 50_000
)

// Entry is one cached response.
type Entry struct {
	Data    json.RawMessage `json:"data"`
	Status  int             `json:"status"`
	Expires int64           `json:"expires"` // unix ms
	ETag    string          `json:"etag,omitempty"`
}

// Fresh reports whether the entry may be served without asking ESI.
func (e *Entry) Fresh(now time.Time) bool {
	return e != nil && e.Expires > now.UnixMilli()
}

// Cache is the two-tier response cache. Safe for concurrent use.
type Cache struct {
	redis *redis.Client

	mu    sync.Mutex
	local map[string]*list.Element
	order list.List
}

type localCacheEntry struct {
	url   string
	entry *Entry
}

// NewCache builds a cache over the shared Valkey.
func NewCache(client *redis.Client) *Cache {
	return &Cache{redis: client, local: make(map[string]*list.Element, 1024)}
}

// Get returns whatever is known about a URL, fresh or not. A stale entry is
// still returned so the caller can use its ETag.
func (c *Cache) Get(ctx context.Context, url string) *Entry {
	c.mu.Lock()
	element := c.local[url]
	var hot *Entry
	if element != nil {
		c.order.MoveToFront(element)
		hot = cloneEntry(element.Value.(*localCacheEntry).entry)
	}
	c.mu.Unlock()
	if hot.Fresh(time.Now()) {
		return hot
	}

	if c.redis == nil {
		return hot
	}
	raw, err := c.redis.Get(ctx, cacheKeyPrefix+hashURL(url)).Bytes()
	if err != nil {
		// A cache miss and a cache outage are the same thing to the caller:
		// proceed to ESI. Returning the stale local entry keeps the ETag.
		return hot
	}

	var entry Entry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return hot
	}
	c.putLocal(url, &entry)
	return cloneEntry(&entry)
}

// Set stores a response in both tiers.
func (c *Cache) Set(ctx context.Context, url string, entry *Entry) {
	if entry == nil {
		return
	}
	c.putLocal(url, entry)

	ttl := time.Until(time.UnixMilli(entry.Expires))
	ttl = max(minCacheTTL, min(maxCacheTTL, ttl))

	payload, err := json.Marshal(entry)
	if err != nil {
		return
	}
	// A failed cache write is not a failed request. The response is already in
	// hand and the next caller simply pays for its own fetch.
	if c.redis != nil {
		_ = c.redis.Set(ctx, cacheKeyPrefix+hashURL(url), payload, ttl).Err()
	}
}

// Touch extends an entry's life after ESI answered 304, keeping the body it
// already had.
func (c *Cache) Touch(ctx context.Context, url string, expires int64, etag string) {
	existing := c.Get(ctx, url)
	if existing == nil {
		return
	}
	updated := *existing
	updated.Expires = expires
	if etag != "" {
		updated.ETag = etag
	}
	c.Set(ctx, url, &updated)
}

func (c *Cache) putLocal(url string, entry *Entry) {
	if entry == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if element := c.local[url]; element != nil {
		element.Value.(*localCacheEntry).entry = cloneEntry(entry)
		c.order.MoveToFront(element)
		return
	}
	if c.order.Len() >= localCacheMax {
		oldest := c.order.Back()
		delete(c.local, oldest.Value.(*localCacheEntry).url)
		c.order.Remove(oldest)
	}
	element := c.order.PushFront(&localCacheEntry{
		url:   url,
		entry: cloneEntry(entry),
	})
	c.local[url] = element
}

func cloneEntry(entry *Entry) *Entry {
	if entry == nil {
		return nil
	}
	cloned := *entry
	cloned.Data = append(json.RawMessage(nil), entry.Data...)
	return &cloned
}

// ParseExpires reads an HTTP Expires header into unix milliseconds, returning
// false when it is missing or unparseable.
func ParseExpires(header string) (int64, bool) {
	if header == "" {
		return 0, false
	}
	t, err := http.ParseTime(header)
	if err != nil {
		return 0, false
	}
	return t.UnixMilli(), true
}
