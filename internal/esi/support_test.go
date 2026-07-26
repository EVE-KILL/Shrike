package esi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// The ESI client is mostly coordination, and coordination is exactly what
// mocking cannot test. A fake Redis that returns whatever the test wants would
// prove nothing about whether the Lua actually stops two workers spending the
// same token, which is the only property that matters.
//
// So these tests use a real Redis and a real HTTP server. Both are cheap: the
// local docker stack is already running for development, and httptest is
// in-process. Tests skip when Redis is unreachable, so `go test ./...` still
// passes with docker down.
//
// They operate on the `esi:*` keyspace of whatever Redis they are pointed at and
// clear it as they go, so point TEST_REDIS_ADDR at a throwaway instance rather
// than something you care about.

const defaultTestRedis = "127.0.0.1:6379"

// testRedisDB isolates this package's tests onto their own Redis database.
//
// `go test ./...` runs different packages in parallel, and these tests clear the
// whole `esi:*` keyspace between cases. Sharing a database with another
// package's tests means one suite deleting another's locks mid-hold — which
// surfaced as an impossible-looking failure where two workers appeared to hold a
// mutex at once. Every package that touches this keyspace picks a distinct
// database; see internal/entities.
const testRedisDB = 15

// reachable caches whether Redis answered, so a run without it pays the dial
// timeout once rather than once per test. Twenty-odd tests each waiting to be
// refused is fifteen seconds of doing nothing.
var reachable = sync.OnceValues(func() (string, error) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = defaultTestRedis
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr, DB: testRedisDB,
		// One attempt and a fast verdict, not five backed-off retries.
		MaxRetries:  -1,
		DialTimeout: 2 * time.Second,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return addr, client.Ping(ctx).Err()
})

func testRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr, err := reachable()
	if err != nil {
		t.Skipf("no test redis at %s: %v", addr, err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr, DB: testRedisDB,
		MaxRetries:  -1,
		DialTimeout: 2 * time.Second,
	})
	t.Cleanup(func() { client.Close() })
	return client
}

// clearESIState removes every key the client owns.
//
// Called before and after each test: state left behind by one test — a paused
// flag especially — silently changes the meaning of the next.
func clearESIState(t *testing.T, rdb *redis.Client) {
	t.Helper()
	ctx := context.Background()

	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, "esi:*", 500).Result()
		if err != nil {
			t.Fatalf("scan esi keys: %v", err)
		}
		if len(keys) > 0 {
			if err := rdb.Del(ctx, keys...).Err(); err != nil {
				t.Fatalf("delete esi keys: %v", err)
			}
		}
		if next == 0 {
			return
		}
		cursor = next
	}
}

// testClient wires a client to a real Redis and a fake ESI.
func testClient(t *testing.T, rdb *redis.Client, baseURL string) *Client {
	t.Helper()
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	return &Client{
		BaseURL:   baseURL,
		UserAgent: "shrike-test/1.0",
		HTTP:      &http.Client{Timeout: 5 * time.Second},
		// The retry ladder is exercised, not waited out: what matters is how
		// many attempts a status earns, not how long the pauses between them
		// are. The production intervals are asserted separately.
		backoff: []time.Duration{time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond},
		limiter: NewRateLimiter(rdb),
		coord:   NewCoordination(rdb),
		cache:   NewCache(rdb),
		coordis: rdb,
	}
}

// fakeESI is an HTTP server standing in for ESI, counting what it was asked.
type fakeESI struct {
	*httptest.Server
	hits   *counter
	handle func(w http.ResponseWriter, r *http.Request)
}

func newFakeESI(t *testing.T, handle func(w http.ResponseWriter, r *http.Request)) *fakeESI {
	t.Helper()

	f := &fakeESI{hits: &counter{}, handle: handle}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.inc()
		f.handle(w, r)
	}))
	t.Cleanup(f.Server.Close)
	return f
}

// Hits reports how many requests reached the server, which is how the cache and
// singleflight tests assert what they actually assert: not "did I get data" but
// "was ESI spared".
func (f *fakeESI) Hits() int { return f.hits.get() }

// jsonOK writes a cacheable 200.
func jsonOK(w http.ResponseWriter, body string, expires time.Time) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Expires", expires.UTC().Format(http.TimeFormat))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}
