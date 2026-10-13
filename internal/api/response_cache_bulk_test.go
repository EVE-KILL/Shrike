package api

import (
	"context"
	"testing"
	"time"
)

type bulkCacheTestBackend struct {
	*fakeResponseCacheBackend
	batches int
	keys    []string
}

func (b *bulkCacheTestBackend) LoadMany(_ context.Context, keys []string) map[string]cachedResponseWithTTL {
	b.batches++
	b.keys = append([]string(nil), keys...)
	result := map[string]cachedResponseWithTTL{}
	for _, key := range keys {
		if value, ok := b.entries[key]; ok {
			result[key] = cachedResponseWithTTL{Entry: cloneCachedResponse(value.response), TTL: value.ttl}
		}
	}
	return result
}
func TestResponseCacheBulkReadsPromoteCloneAndPreserveInvalidation(t *testing.T) {
	shared := &bulkCacheTestBackend{fakeResponseCacheBackend: newFakeResponseCacheBackend()}
	ctx := context.Background()
	writer := newResponseCache(shared, 4096)
	reader := newResponseCache(shared, 4096)
	writer.Store(ctx, "one", cachedResponse{Body: []byte("one")}, time.Minute)
	writer.Store(ctx, "two", cachedResponse{Body: []byte("two")}, time.Minute)
	writer.Store(ctx, "battle:123", cachedResponse{Body: []byte("battle")}, time.Minute)
	shared.entries["stale"] = fakeResponseCacheEntry{response: cachedResponse{Body: []byte("stale"), FreshUntil: time.Now().Add(-time.Minute)}, ttl: time.Minute}
	result := reader.LoadMany(ctx, []string{"one", "two", "one", "stale", "missing", "battle:123"})
	if shared.batches != 1 || shared.loads != 0 || len(shared.keys) != 5 || len(result) != 3 {
		t.Fatal("bulk misses must use one round trip, deduplicate keys and omit stale/missing data")
	}
	result["one"].Body[0] = 'x'
	if got, ok := reader.Load(ctx, "one"); !ok || string(got.Body) != "one" {
		t.Fatal("a returned payload mutated cached data")
	}
	writer.DeleteMatching(ctx, "battle:*")
	result = reader.LoadMany(ctx, []string{"one", "two", "battle:123"})
	if len(result) != 2 || shared.batches != 2 || len(shared.keys) != 1 || shared.keys[0] != "battle:123" {
		t.Fatal("promoted immutable data must stay local while mutable data observes shared invalidation")
	}
}

func TestRedisBulkDecoderRejectsExpiredAndMalformedRows(t *testing.T) {
	for _, row := range []struct {
		fields []any
		ttl    time.Duration
	}{{[]any{"application/json", nil, "1"}, time.Minute}, {[]any{"application/json", "{}", "1"}, 0}, {[]any{"application/json", "{}"}, time.Minute}} {
		if _, _, ok := decodeRedisCachedResponse(row.fields, row.ttl); ok {
			t.Fatal("invalid cache row was accepted")
		}
	}
}
