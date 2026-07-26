package images

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

const defaultCacheMaximumEntries = 100_000

type cacheEntry struct {
	key       string
	value     Result
	size      int64
	expiresAt time.Time
	element   *list.Element
}

// Cache is the serve-only, byte-bounded LRU for final encoded responses.
// Values are immutable after insertion, so a hit can be written directly to
// the response without another allocation proportional to the image size.
type Cache struct {
	mu         sync.Mutex
	maxBytes   int64
	maxEntries int
	ttl        time.Duration
	now        func() time.Time
	bytes      int64
	hits       uint64
	misses     uint64
	entries    map[string]*cacheEntry
	order      *list.List
}

type CacheStats struct {
	Entries int
	Bytes   int64
	Maximum int64
	Hits    uint64
	Misses  uint64
}

func NewCache(maxBytes int64, ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &Cache{
		maxBytes: maxBytes, maxEntries: defaultCacheMaximumEntries,
		ttl: ttl, now: time.Now,
		entries: make(map[string]*cacheEntry), order: list.New(),
	}
}

func (c *Cache) Get(key string) (Result, bool) {
	if c == nil || c.maxBytes <= 0 {
		return Result{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		c.misses++
		return Result{}, false
	}
	if !entry.expiresAt.After(c.now()) {
		c.removeLocked(entry)
		c.misses++
		return Result{}, false
	}
	c.order.MoveToFront(entry.element)
	c.hits++
	return entry.value, true
}

func (c *Cache) Put(key string, value Result) {
	if c == nil || c.maxBytes <= 0 || len(value.Body) == 0 {
		return
	}
	size := int64(len(value.Body))
	// One pathological response must not flush the useful hot set.
	if size > max(c.maxBytes/10, 1) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if current := c.entries[key]; current != nil {
		c.removeLocked(current)
	}
	entry := &cacheEntry{
		key: key, value: value, size: size,
		expiresAt: c.now().Add(c.ttl),
	}
	entry.element = c.order.PushFront(entry)
	c.entries[key] = entry
	c.bytes += size
	for c.bytes > c.maxBytes || len(c.entries) > c.maxEntries {
		back := c.order.Back()
		if back == nil {
			break
		}
		oldest, _ := back.Value.(*cacheEntry)
		if oldest == nil {
			break
		}
		c.removeLocked(oldest)
	}
}

func (c *Cache) RemovePrefix(prefix string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.entries {
		if strings.HasPrefix(key, prefix) {
			c.removeLocked(entry)
		}
	}
}

func (c *Cache) Stats() CacheStats {
	if c == nil {
		return CacheStats{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStats{
		Entries: len(c.entries), Bytes: c.bytes, Maximum: c.maxBytes,
		Hits: c.hits, Misses: c.misses,
	}
}

func (c *Cache) removeLocked(entry *cacheEntry) {
	delete(c.entries, entry.key)
	c.order.Remove(entry.element)
	c.bytes -= entry.size
	if c.bytes < 0 {
		c.bytes = 0
	}
}
