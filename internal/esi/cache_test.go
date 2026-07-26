package esi

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestCacheRoundTrip(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	c := NewCache(rdb)
	url := "https://esi.test/" + t.Name()

	entry := &Entry{
		Data:    json.RawMessage(`{"name":"Rifter"}`),
		Status:  200,
		Expires: time.Now().Add(time.Hour).UnixMilli(),
		ETag:    `W/"abc123"`,
	}
	c.Set(ctx, url, entry)

	got := c.Get(ctx, url)
	if got == nil {
		t.Fatal("nothing came back")
	}
	if string(got.Data) != string(entry.Data) {
		t.Errorf("data = %s", got.Data)
	}
	if got.ETag != entry.ETag {
		t.Errorf("etag = %q", got.ETag)
	}
	if !got.Fresh(time.Now()) {
		t.Error("an entry expiring in an hour is not fresh")
	}
}

// A second process must see what the first cached — that is the entire point of
// the shared tier, and the reason ten workers asking for one character produce
// one ESI request rather than ten.
func TestCacheIsSharedAcrossProcesses(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	url := "https://esi.test/" + t.Name()

	writer := NewCache(rdb)
	writer.Set(ctx, url, &Entry{
		Data:    json.RawMessage(`{"shared":true}`),
		Status:  200,
		Expires: time.Now().Add(time.Hour).UnixMilli(),
	})

	// A separate Cache value has an empty local tier, standing in for another
	// process that has never seen this URL.
	reader := NewCache(rdb)
	got := reader.Get(ctx, url)
	if got == nil {
		t.Fatal("a cold reader saw nothing")
	}
	if string(got.Data) != `{"shared":true}` {
		t.Errorf("data = %s", got.Data)
	}
}

// An expired entry is still worth keeping: its ETag buys a 304, which costs no
// error budget and transfers no body. Dropping it on expiry would throw that
// away.
func TestExpiredEntryStillCarriesETag(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	c := NewCache(rdb)
	url := "https://esi.test/" + t.Name()

	c.Set(ctx, url, &Entry{
		Data:    json.RawMessage(`{"old":true}`),
		Status:  200,
		Expires: time.Now().Add(-time.Hour).UnixMilli(),
		ETag:    `W/"stale"`,
	})

	got := c.Get(ctx, url)
	if got == nil {
		t.Fatal("an expired entry was discarded entirely")
	}
	if got.Fresh(time.Now()) {
		t.Error("an expired entry reported itself fresh")
	}
	if got.ETag != `W/"stale"` {
		t.Errorf("the etag was lost: %q", got.ETag)
	}
}

func TestTouchExtendsWithoutLosingBody(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	c := NewCache(rdb)
	url := "https://esi.test/" + t.Name()

	c.Set(ctx, url, &Entry{
		Data:    json.RawMessage(`{"body":"kept"}`),
		Status:  200,
		Expires: time.Now().Add(-time.Minute).UnixMilli(),
		ETag:    `W/"v1"`,
	})

	newExpiry := time.Now().Add(2 * time.Hour).UnixMilli()
	c.Touch(ctx, url, newExpiry, `W/"v2"`)

	got := c.Get(ctx, url)
	if got == nil {
		t.Fatal("the entry vanished")
	}
	if !got.Fresh(time.Now()) {
		t.Error("touch did not make the entry fresh again")
	}
	if string(got.Data) != `{"body":"kept"}` {
		t.Errorf("touch lost the body: %s", got.Data)
	}
	if got.ETag != `W/"v2"` {
		t.Errorf("touch did not take the new etag: %q", got.ETag)
	}
}

// A miss must be a miss, not an empty entry that later reads as a cached
// response with no data.
func TestCacheMissReturnsNil(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	c := NewCache(rdb)
	if got := c.Get(context.Background(), "https://esi.test/never-stored"); got != nil {
		t.Errorf("a miss returned %+v", got)
	}
}

// The local tier is bounded, or a long-running worker's cache grows until the
// process is killed.
func TestLocalTierIsBounded(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	c := NewCache(rdb)
	entry := &Entry{Status: 200, Expires: time.Now().Add(time.Hour).UnixMilli()}

	for i := 0; i < localCacheMax+100; i++ {
		c.putLocal(string(rune(i))+"-url", entry)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.local) > localCacheMax {
		t.Errorf("local tier holds %d entries, above the %d cap", len(c.local), localCacheMax)
	}
	if len(c.order) > localCacheMax {
		t.Errorf("eviction list holds %d keys, above the %d cap", len(c.order), localCacheMax)
	}
}
