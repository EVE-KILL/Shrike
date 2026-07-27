package api

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeResponseCacheBackend struct {
	mu      sync.Mutex
	entries map[string]fakeResponseCacheEntry
	loads   int
	stores  int
	deletes int
}

type fakeResponseCacheEntry struct {
	response cachedResponse
	ttl      time.Duration
}

func newFakeResponseCacheBackend() *fakeResponseCacheBackend {
	return &fakeResponseCacheBackend{
		entries: make(map[string]fakeResponseCacheEntry),
	}
}

func (f *fakeResponseCacheBackend) Load(
	_ context.Context,
	key string,
) (cachedResponse, time.Duration, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loads++
	entry, ok := f.entries[key]
	return cloneCachedResponse(entry.response), entry.ttl, ok
}

func (f *fakeResponseCacheBackend) Store(
	_ context.Context,
	key string,
	response cachedResponse,
	ttl time.Duration,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stores++
	f.entries[key] = fakeResponseCacheEntry{
		response: cloneCachedResponse(response),
		ttl:      ttl,
	}
}

func (f *fakeResponseCacheBackend) DeleteMatching(
	_ context.Context,
	pattern string,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletes++
	match := cacheGlob(pattern)
	for key := range f.entries {
		if match.MatchString(key) {
			delete(f.entries, key)
		}
	}
}

func TestResponseCacheWritesThroughAndServesL1First(t *testing.T) {
	shared := newFakeResponseCacheBackend()
	cache := newResponseCache(shared, 1024)
	entry := cachedResponse{
		ContentType: "application/json",
		Body:        []byte(`{"source":"database"}`),
	}

	cache.Store(context.Background(), "response", entry, time.Minute)
	got, ok := cache.Load(context.Background(), "response")
	if !ok || string(got.Body) != string(entry.Body) {
		t.Fatalf("L1 response = %q, %v", got.Body, ok)
	}
	if shared.stores != 1 {
		t.Fatalf("L2 stores = %d, want 1", shared.stores)
	}
	if shared.loads != 0 {
		t.Fatalf("L2 loads = %d, want 0 for an L1 hit", shared.loads)
	}
}

func TestResponseCachePromotesL2HitIntoL1(t *testing.T) {
	shared := newFakeResponseCacheBackend()
	shared.entries["response"] = fakeResponseCacheEntry{
		response: cachedResponse{
			ContentType: "application/json",
			Body:        []byte(`{"source":"valkey"}`),
		},
		ttl: time.Minute,
	}
	cache := newResponseCache(shared, 1024)

	for i := 0; i < 2; i++ {
		got, ok := cache.Load(context.Background(), "response")
		if !ok || string(got.Body) != `{"source":"valkey"}` {
			t.Fatalf("load %d = %q, %v", i, got.Body, ok)
		}
	}
	if shared.loads != 1 {
		t.Fatalf("L2 loads = %d, want 1 after promotion", shared.loads)
	}
}

func TestResponseLRUEvictsLeastRecentlyUsed(t *testing.T) {
	now := time.Now()
	response := cachedResponse{
		ContentType: "text/plain",
		Body:        []byte("same-size"),
	}
	size := cachedResponseSize("a", response)
	cache := newResponseLRU(2 * size)

	cache.Put("a", response, time.Minute, now)
	cache.Put("b", response, time.Minute, now)
	if _, ok := cache.Get("a", now); !ok {
		t.Fatal("entry a was not present before eviction")
	}
	cache.Put("c", response, time.Minute, now)

	if _, ok := cache.Get("a", now); !ok {
		t.Error("recently read entry a was evicted")
	}
	if _, ok := cache.Get("b", now); ok {
		t.Error("least recently used entry b survived")
	}
	if _, ok := cache.Get("c", now); !ok {
		t.Error("new entry c was not retained")
	}
}

func TestResponseLRUDoesNotOutliveTTL(t *testing.T) {
	now := time.Now()
	cache := newResponseLRU(1024)
	cache.Put("response", cachedResponse{
		ContentType: "application/json",
		Body:        []byte(`{"fresh":true}`),
	}, time.Minute, now)

	if _, ok := cache.Get("response", now.Add(time.Minute)); ok {
		t.Error("entry survived its time to live")
	}
}

func TestResponseCacheNormalizesContentTypeInBothTiers(t *testing.T) {
	shared := newFakeResponseCacheBackend()
	cache := newResponseCache(shared, 1024)
	cache.Store(context.Background(), "response", cachedResponse{
		Body: []byte(`{"ok":true}`),
	}, time.Minute)

	got, ok := cache.Load(context.Background(), "response")
	if !ok || got.ContentType != "application/json" {
		t.Fatalf("L1 content type = %q, %v", got.ContentType, ok)
	}
	if got := shared.entries["response"].response.ContentType; got !=
		"application/json" {
		t.Fatalf("L2 content type = %q", got)
	}
}

func TestResponseCacheDeleteMatchingInvalidatesBothTiers(t *testing.T) {
	shared := newFakeResponseCacheBackend()
	cache := newResponseCache(shared, 4096)
	for _, key := range []string{
		"shrike:web-api:dev:battles",
		"shrike:web-api:dev:campaigns",
	} {
		cache.Store(context.Background(), key, cachedResponse{
			ContentType: "application/json",
			Body:        []byte(key),
		}, time.Minute)
	}

	cache.DeleteMatching(context.Background(), "shrike:web-api:*:*battle*")
	if _, ok := cache.Load(
		context.Background(),
		"shrike:web-api:dev:battles",
	); ok {
		t.Error("matching entry survived invalidation")
	}
	if _, ok := cache.Load(
		context.Background(),
		"shrike:web-api:dev:campaigns",
	); !ok {
		t.Error("non-matching entry was invalidated")
	}
	if shared.deletes != 1 {
		t.Fatalf("L2 invalidations = %d, want 1", shared.deletes)
	}
}
