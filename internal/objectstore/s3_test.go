package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestS3StoreReadCacheEvictsLeastRecentlyUsedWithinByteLimit(
	t *testing.T,
) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	store, err := NewS3Store(S3Options{
		Endpoint: "https://s3.example.test", Bucket: "media",
		Region: "region-1", AccessKeyID: "id", SecretAccessKey: "secret",
		CacheTTL: time.Hour, CacheMaximumBytes: 6,
		now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	store.cachePut("a", []byte("aaa"))
	store.cachePut("b", []byte("bb"))
	if _, ok := store.cacheGet("a"); !ok {
		t.Fatal("recently used entry was not cached")
	}
	store.cachePut("c", []byte("cc"))

	store.cacheMu.Lock()
	_, hasA := store.cache["a"]
	_, hasB := store.cache["b"]
	_, hasC := store.cache["c"]
	cacheBytes := store.cacheBytes
	cacheEntries := len(store.cache)
	mostRecent, _ := store.cacheOrder.Front().Value.(string)
	leastRecent, _ := store.cacheOrder.Back().Value.(string)
	store.cacheMu.Unlock()

	if !hasA || hasB || !hasC {
		t.Fatalf(
			"cache entries: a=%t b=%t c=%t",
			hasA,
			hasB,
			hasC,
		)
	}
	if cacheBytes != 5 || cacheEntries != 2 {
		t.Fatalf(
			"cache size = %d bytes across %d entries",
			cacheBytes,
			cacheEntries,
		)
	}
	if mostRecent != "c" || leastRecent != "a" {
		t.Fatalf(
			"LRU order = newest %q, oldest %q",
			mostRecent,
			leastRecent,
		)
	}
}

func TestS3StoreReadCacheSkipsOversizedEntriesWithoutServingStaleData(
	t *testing.T,
) {
	store, err := NewS3Store(S3Options{
		Endpoint: "https://s3.example.test", Bucket: "media",
		Region: "region-1", AccessKeyID: "id", SecretAccessKey: "secret",
		CacheMaximumBytes: 4,
	})
	if err != nil {
		t.Fatal(err)
	}

	store.cachePut("asset", []byte("old"))
	store.cachePut("asset", []byte("newer"))

	if body, ok := store.cacheGet("asset"); ok {
		t.Fatalf("oversized replacement left cached data %q", body)
	}
	store.cacheMu.Lock()
	cacheBytes := store.cacheBytes
	cacheEntries := len(store.cache)
	orderEntries := store.cacheOrder.Len()
	store.cacheMu.Unlock()
	if cacheBytes != 0 || cacheEntries != 0 || orderEntries != 0 {
		t.Fatalf(
			"cache after oversized replacement = %d bytes, %d map entries, %d LRU entries",
			cacheBytes,
			cacheEntries,
			orderEntries,
		)
	}
}

func TestS3StoreReadCacheRemovesExpiredEntryAccounting(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	store, err := NewS3Store(S3Options{
		Endpoint: "https://s3.example.test", Bucket: "media",
		Region: "region-1", AccessKeyID: "id", SecretAccessKey: "secret",
		CacheTTL: time.Minute, CacheMaximumBytes: 4,
		now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	store.cachePut("asset", []byte("1234"))
	now = now.Add(time.Minute)
	if body, ok := store.cacheGet("asset"); ok {
		t.Fatalf("expired cache entry returned %q", body)
	}
	store.cacheMu.Lock()
	cacheBytes := store.cacheBytes
	cacheEntries := len(store.cache)
	orderEntries := store.cacheOrder.Len()
	store.cacheMu.Unlock()
	if cacheBytes != 0 || cacheEntries != 0 || orderEntries != 0 {
		t.Fatalf(
			"expired cache accounting = %d bytes, %d map entries, %d LRU entries",
			cacheBytes,
			cacheEntries,
			orderEntries,
		)
	}
}

func TestS3StoreReadCacheBoundsEmptyEntries(t *testing.T) {
	store, err := NewS3Store(S3Options{
		Endpoint: "https://s3.example.test", Bucket: "media",
		Region: "region-1", AccessKeyID: "id", SecretAccessKey: "secret",
		CacheMaximumBytes: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	for index := 0; index <= defaultCacheMaximumEntries; index++ {
		store.cachePut(fmt.Sprintf("empty-%04d", index), nil)
	}
	store.cacheMu.Lock()
	cacheEntries := len(store.cache)
	orderEntries := store.cacheOrder.Len()
	_, oldestSurvived := store.cache["empty-0000"]
	_, newestSurvived := store.cache[fmt.Sprintf(
		"empty-%04d",
		defaultCacheMaximumEntries,
	)]
	store.cacheMu.Unlock()

	if cacheEntries != defaultCacheMaximumEntries ||
		orderEntries != defaultCacheMaximumEntries {
		t.Fatalf(
			"empty cache entries = %d map, %d LRU",
			cacheEntries,
			orderEntries,
		)
	}
	if oldestSurvived || !newestSurvived {
		t.Fatalf(
			"empty-entry eviction = oldest %t, newest %t",
			oldestSurvived,
			newestSurvived,
		)
	}
}

func TestS3StoreSignsCRUDAndCachesReads(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []string
		stored   = []byte("from-server")
	)
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.EscapedPath())
		mu.Unlock()
		if !strings.Contains(
			r.Header.Get("Authorization"),
			"Credential=test-key/",
		) || !strings.Contains(
			r.Header.Get("Authorization"),
			"/eu-central-003/s3/aws4_request",
		) {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Amz-Date") == "" ||
			r.Header.Get("X-Amz-Content-Sha256") == "" {
			t.Errorf("request lacks signed S3 headers: %#v", r.Header)
		}
		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			if string(body) != "uploaded" ||
				r.Header.Get("Content-Type") != "image/png" {
				t.Errorf("PUT = %q, %q", body, r.Header.Get("Content-Type"))
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			_, _ = w.Write(stored)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store, err := NewS3Store(S3Options{
		Endpoint: server.URL + "/s3", Bucket: "media",
		Region: BackblazeRegion, AccessKeyID: "test-key",
		SecretAccessKey: "test-secret", CacheTTL: time.Hour,
		now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	const key = "domains/7/banner_11"
	if err := store.Put(
		context.Background(), key, []byte("uploaded"), "image/png",
	); err != nil {
		t.Fatal(err)
	}
	first, err := store.Get(context.Background(), key)
	if err != nil || string(first) != "uploaded" {
		t.Fatalf("cached GET = %q, %v", first, err)
	}
	first[0] = 'X'
	second, err := store.Get(context.Background(), key)
	if err != nil || string(second) != "uploaded" {
		t.Fatalf("cache returned mutable storage = %q, %v", second, err)
	}
	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	afterDelete, err := store.Get(context.Background(), key)
	if err != nil || string(afterDelete) != string(stored) {
		t.Fatalf("GET after eviction = %q, %v", afterDelete, err)
	}

	mu.Lock()
	gotRequests := append([]string(nil), requests...)
	mu.Unlock()
	want := []string{
		"PUT /s3/media/domains/7/banner_11",
		"DELETE /s3/media/domains/7/banner_11",
		"GET /s3/media/domains/7/banner_11",
	}
	if len(gotRequests) != len(want) {
		t.Fatalf("requests = %v, want %v", gotRequests, want)
	}
	for i := range want {
		if gotRequests[i] != want[i] {
			t.Errorf("request %d = %q, want %q", i, gotRequests[i], want[i])
		}
	}
}

func TestS3StoreNotFoundAndStatusErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if strings.Contains(r.URL.Path, "missing") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "bucket unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	store, err := NewS3Store(S3Options{
		Endpoint: server.URL, Bucket: "media", Region: BackblazeRegion,
		AccessKeyID: "id", SecretAccessKey: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := store.Get(context.Background(), "missing")
	if err != nil || body != nil {
		t.Fatalf("missing object = %q, %v", body, err)
	}
	if _, err := store.Get(context.Background(), "broken"); err == nil ||
		!strings.Contains(err.Error(), "503") {
		t.Fatalf("status error = %v", err)
	}
	if err := store.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}
}

func TestS3StoreStatMetadataAndUncachedObjects(t *testing.T) {
	var gets int
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("ETag", `"abc123"`)
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			w.Header().Set("Content-Length", "7")
			w.Header().Set("Last-Modified", "Sun, 26 Jul 2026 12:00:00 GMT")
			w.Header().Set("X-Amz-Meta-Origin-Etag", "ccp-value")
		case http.MethodGet:
			gets++
			w.Header().Set("ETag", `"abc123"`)
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("payload"))
		case http.MethodPut:
			if got := r.Header.Get("X-Amz-Meta-Origin-Etag"); got != "ccp-value" {
				t.Errorf("origin metadata = %q", got)
			}
			if got := r.Header.Get("Cache-Control"); got != "public, max-age=60" {
				t.Errorf("cache control = %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	store, err := NewS3Store(S3Options{
		Endpoint: server.URL, Bucket: "images", Region: BackblazeRegion,
		AccessKeyID: "id", SecretAccessKey: "secret",
		DisableCache: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	info, err := store.Stat(ctx, "entities/character/7/original")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.ETag != "abc123" || info.Size != 7 ||
		info.ContentType != "image/png" ||
		info.Metadata["origin-etag"] != "ccp-value" {
		t.Fatalf("stat = %#v", info)
	}

	for range 2 {
		object, getErr := store.GetObject(ctx, "entities/character/7/original")
		if getErr != nil || object == nil || string(object.Body) != "payload" {
			t.Fatalf("GetObject = %#v, %v", object, getErr)
		}
	}
	if gets != 2 {
		t.Fatalf("uncached GET count = %d, want 2", gets)
	}

	if err := store.PutWithOptions(
		ctx,
		"entities/character/7/original",
		[]byte("payload"),
		PutOptions{
			ContentType:  "image/png",
			CacheControl: "public, max-age=60",
			Metadata:     map[string]string{"origin-etag": "ccp-value"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestS3StoreRetriesTransientResponsesWithTheOriginalBody(t *testing.T) {
	var attempts int
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		attempts++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, string(body))
		if attempts < 3 {
			http.Error(w, "temporary B2 incident", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var delays []time.Duration
	store, err := NewS3Store(S3Options{
		Endpoint: server.URL, Bucket: "images", Region: BackblazeRegion,
		AccessKeyID: "id", SecretAccessKey: "secret",
		DisableCache: true,
		sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(
		context.Background(),
		"static/systems/30000551.png",
		[]byte("image"),
		"image/png",
	); err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	for attempt, body := range bodies {
		if body != "image" {
			t.Errorf("attempt %d body = %q", attempt+1, body)
		}
	}
	if len(delays) != 2 ||
		delays[0] != defaultRetryDelay ||
		delays[1] != 2*defaultRetryDelay {
		t.Errorf("retry delays = %v", delays)
	}
}

func TestObjectStoreRetryDelayHonorsRetryAfterAndCapsIt(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	response := &http.Response{Header: make(http.Header)}
	response.Header.Set("Retry-After", "60")
	if got := objectStoreRetryDelay(0, response, now); got != maximumRetryDelay {
		t.Errorf("seconds Retry-After delay = %s", got)
	}
	response.Header.Set("Retry-After", now.Add(2*time.Second).Format(http.TimeFormat))
	if got := objectStoreRetryDelay(0, response, now); got != 2*time.Second {
		t.Errorf("date Retry-After delay = %s", got)
	}
}

func TestS3StoreRejectsUnsafeConfigurationKeysAndLargeObjects(t *testing.T) {
	valid := S3Options{
		Endpoint: "https://s3.example.test", Bucket: "media",
		Region: "region-1", AccessKeyID: "id", SecretAccessKey: "secret",
		MaximumBytes: 4,
	}
	for name, mutate := range map[string]func(*S3Options){
		"endpoint": func(options *S3Options) { options.Endpoint = "file:///tmp/store" },
		"bucket":   func(options *S3Options) { options.Bucket = "../media" },
		"region":   func(options *S3Options) { options.Region = "" },
		"key":      func(options *S3Options) { options.AccessKeyID = "" },
		"cache size": func(options *S3Options) {
			options.CacheMaximumBytes = -1
		},
	} {
		t.Run(name, func(t *testing.T) {
			options := valid
			mutate(&options)
			if _, err := NewS3Store(options); err == nil {
				t.Fatal("invalid options were accepted")
			}
		})
	}

	store, err := NewS3Store(valid)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "/absolute", "a//b", "a/../b", `a\b`} {
		if _, err := store.objectURL(key); err == nil {
			t.Errorf("unsafe key %q was accepted", key)
		}
	}
	if err := store.Put(
		context.Background(), "safe", []byte("12345"), "text/plain",
	); err == nil {
		t.Error("oversize PUT reached the network")
	}
	if err := store.PutWithOptions(
		context.Background(),
		"safe",
		[]byte("1234"),
		PutOptions{Metadata: map[string]string{"bad:name": "value"}},
	); err == nil {
		t.Error("unsafe metadata reached the network")
	}
}
