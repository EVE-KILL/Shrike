package api

import (
	"context"
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
