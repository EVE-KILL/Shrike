package images

import (
	"fmt"
	"testing"
	"time"
)

func TestCacheBoundsBytesUsesLRUAndExpires(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	cache := NewCache(100, time.Minute)
	cache.now = func() time.Time { return now }

	for index := range 10 {
		cache.Put(fmt.Sprintf("%02d", index), Result{Body: make([]byte, 10)})
	}
	if _, ok := cache.Get("00"); !ok {
		t.Fatal("oldest entry was not initially cached")
	}
	cache.Put("10", Result{Body: make([]byte, 10)})
	if _, ok := cache.Get("01"); ok {
		t.Fatal("least-recently-used entry survived byte eviction")
	}
	if _, ok := cache.Get("00"); !ok {
		t.Fatal("recently used entry was evicted")
	}

	now = now.Add(time.Minute)
	if _, ok := cache.Get("00"); ok {
		t.Fatal("expired entry was returned")
	}
	stats := cache.Stats()
	if stats.Bytes < 0 || stats.Entries != 9 {
		t.Fatalf("cache stats after expiry = %+v", stats)
	}
}

func TestCacheSkipsSingleOversizedResponse(t *testing.T) {
	cache := NewCache(100, time.Hour)
	cache.Put("large", Result{Body: make([]byte, 11)})
	if _, ok := cache.Get("large"); ok {
		t.Fatal("response larger than ten percent of the cache was retained")
	}
}
