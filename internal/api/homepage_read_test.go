package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestHomepageDataCacheReusesTypedReads(t *testing.T) {
	reader := newCachedHomepageReader(nil, nil, newResponseCache(nil, 1<<20), "test")
	loads := 0
	load := func(context.Context) ([]TopEntry, error) {
		loads++
		return []TopEntry{{ID: 42, Name: "Example", Count: 12}}, nil
	}
	for range 2 {
		entries, err := cachedHomepage(context.Background(), reader, "top:characters:7:10", time.Minute, load)
		if err != nil || len(entries) != 1 || entries[0].Name != "Example" {
			t.Fatalf("cached entries = %#v, error = %v", entries, err)
		}
	}
	if loads != 1 {
		t.Fatalf("source loads = %d, want one", loads)
	}
}

func TestCachedKillsTopListIsScopedByType(t *testing.T) {
	cache := newResponseCache(nil, 1<<20)
	reader := newCachedHomepageReader(nil, nil, cache, "test")
	body, err := json.Marshal([]TopEntry{{ID: 42, Name: "Example", Count: 7}})
	if err != nil {
		t.Fatal(err)
	}
	cache.StoreStale(context.Background(), reader.cacheKey(killsTopCacheName("big", "corporations", 7, 10)),
		cachedResponse{Body: body}, 5*time.Minute, 5*time.Minute)
	if entries, ok := reader.TopKillsCached(context.Background(), "big", "corporations", 7, 10); !ok ||
		len(entries) != 1 || entries[0].Name != "Example" {
		t.Fatalf("cached big corporations = %#v, %t", entries, ok)
	}
	if _, ok := reader.TopKillsCached(context.Background(), "solo", "corporations", 7, 10); ok {
		t.Fatal("solo top list reused big-kills cache entry")
	}
}
