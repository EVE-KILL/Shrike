package queue

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Everything that needs Postgres or Redis skips when they are unreachable, so
// the whole package still runs on a laptop with nothing started and in a CI job
// that has not brought the stack up yet.

const (
	queueTestDSN = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"

	// A dedicated Redis database, because `go test ./...` runs packages in
	// parallel and these tests clear the TQ key. Sharing one keyspace with
	// internal/esi produced a phantom failure once already.
	queueTestRedisDB = 13
)

// poolOnce caches the availability check. Without it every skipped test pays
// the dial timeout again, which turns "no database" from instant into slow.
var (
	poolOnce  sync.Once
	sharedDSN string
	poolErr   error
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	poolOnce.Do(func() {
		sharedDSN = os.Getenv("TEST_DATABASE_URL")
		if sharedDSN == "" {
			sharedDSN = queueTestDSN
		}

		pool, err := pgxpool.New(context.Background(), sharedDSN)
		if err != nil {
			poolErr = err
			return
		}
		defer pool.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		poolErr = pool.Ping(ctx)
	})

	if poolErr != nil {
		t.Skipf("no test database: %v", poolErr)
	}

	pool, err := pgxpool.New(context.Background(), sharedDSN)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)

	// River's schema has to be there. A test that silently passed against an
	// unmigrated database would be asserting nothing.
	state, err := MigrationState(context.Background(), pool)
	if err != nil || !state.UpToDate() {
		t.Skipf("River schema is not migrated — run `shrike queue:migrate` (%v)", err)
	}
	return pool
}

func testRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	// MaxRetries -1 so an absent Redis is reported at once rather than after
	// five redials. Every skipping test pays this, so it is the difference
	// between the package skipping instantly and taking seconds to do nothing.
	rdb := redis.NewClient(&redis.Options{
		Addr:        addr,
		DB:          queueTestRedisDB,
		DialTimeout: time.Second,
		MaxRetries:  -1,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		t.Skipf("no test redis: %v", err)
	}

	t.Cleanup(func() {
		_ = rdb.Del(context.Background(), TQStatusKey).Err()
		_ = rdb.Close()
	})
	return rdb
}

// testQueue is a queue name no declaration uses, so these tests cannot disturb
// a real backlog on a shared development database.
const testQueue = "shrike_test_queue"

// clearTestJobs removes anything a previous run left behind.
func clearTestJobs(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	del := func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM river_job WHERE queue = $1 OR kind LIKE 'shrike_test_%'`, testQueue)
	}
	del()
	t.Cleanup(del)
}
