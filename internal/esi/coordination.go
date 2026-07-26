package esi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cluster-wide coordination: two primitives, both self-expiring.
//
//  1. Singleflight. When every worker processing a battle asks for the same
//     character at the same moment, one of them fetches and the rest wait for
//     the cache to fill. Without it, a fleet fight turns into hundreds of
//     identical ESI requests.
//
//  2. A sequential lock per group, for the families that 420 under a concurrent
//     burst regardless of the token bucket.
//
// Every lock has a TTL, so a worker that dies mid-request cannot wedge the
// cluster. Release is fenced by a random token, so a worker whose lock already
// expired cannot delete the next owner's.

const (
	inflightPrefix = "esi:v2:inflight:"
	seqPrefix      = "esi:v2:seq:"
)

const (
	// claimTTL must exceed the worst-case request, retries included, or a slow
	// fetch loses its claim and a second worker starts the same request.
	claimTTL = 45 * time.Second
	// claimWaitMax bounds how long a loser waits before giving up and going its
	// own way.
	claimWaitMax = 30 * time.Second
	// claimPoll trades Redis load against how quickly a waiter notices.
	claimPoll = 50 * time.Millisecond

	// seqLockTTL is shorter than claimTTL: these groups are paced so slowly that
	// any single request completes quickly, and a long TTL would stall the queue
	// behind a dead holder.
	seqLockTTL  = 30 * time.Second
	seqWaitMax  = 60 * time.Second
	seqBackoff0 = 25 * time.Millisecond
	seqBackoffX = 500 * time.Millisecond
)

// luaReleaseIfMatch deletes a lock only when the caller still owns it.
const luaReleaseIfMatch = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
else
    return 0
end
`

var releaseScript = redis.NewScript(luaReleaseIfMatch)

// Coordination provides the singleflight and sequential primitives.
type Coordination struct {
	redis *redis.Client
}

// NewCoordination builds it over the coordination instance.
func NewCoordination(client *redis.Client) *Coordination {
	return &Coordination{redis: client}
}

// TryClaim attempts to become the one worker fetching url. An empty token means
// somebody else already is.
func (c *Coordination) TryClaim(ctx context.Context, url string) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	ok, err := c.redis.SetArgs(ctx, inflightPrefix+hashURL(url), token, redis.SetArgs{
		Mode: "NX",
		TTL:  claimTTL,
	}).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if ok != "OK" {
		return "", nil
	}
	return token, nil
}

// ReleaseClaim gives up ownership. Failing to release is not worth surfacing:
// the TTL cleans up regardless, and the caller is usually already unwinding
// from a more interesting error.
func (c *Coordination) ReleaseClaim(ctx context.Context, url, token string) {
	if token == "" {
		return
	}
	_ = releaseScript.Run(ctx, c.redis, []string{inflightPrefix + hashURL(url)}, token).Err()
}

// WaitForClaim blocks until the current holder finishes or its claim expires.
// A false return means the wait timed out; the caller should try the request
// itself rather than assume the cache was filled.
func (c *Coordination) WaitForClaim(ctx context.Context, url string) bool {
	key := inflightPrefix + hashURL(url)
	deadline := time.Now().Add(claimWaitMax)

	ticker := time.NewTicker(claimPoll)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		n, err := c.redis.Exists(ctx, key).Result()
		if err != nil || n == 0 {
			return err == nil
		}
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
	return false
}

// AcquireSequential waits in line for a group's mutex. An empty token means the
// wait timed out.
func (c *Coordination) AcquireSequential(ctx context.Context, group string) (string, error) {
	key := seqPrefix + group
	deadline := time.Now().Add(seqWaitMax)
	backoff := seqBackoff0

	for time.Now().Before(deadline) {
		token, err := randomToken()
		if err != nil {
			return "", err
		}
		res, err := c.redis.SetArgs(ctx, key, token, redis.SetArgs{Mode: "NX", TTL: seqLockTTL}).Result()
		if err == nil && res == "OK" {
			return token, nil
		}
		if err != nil && err != redis.Nil {
			return "", err
		}

		// Jittered backoff. Without the jitter, a queue of waiters wakes in
		// lockstep and every retry collides with every other.
		jitter := time.Duration(randInt63n(int64(backoff / 3)))
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(backoff + jitter):
		}
		backoff = min(seqBackoffX, backoff*3/2)
	}
	return "", nil
}

// ReleaseSequential hands the group's mutex to the next waiter.
func (c *Coordination) ReleaseSequential(ctx context.Context, group, token string) {
	if token == "" {
		return
	}
	_ = releaseScript.Run(ctx, c.redis, []string{seqPrefix + group}, token).Err()
}

func hashURL(url string) string {
	sum := sha256.Sum256([]byte(url))
	return hex.EncodeToString(sum[:])[:32]
}

func randomToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// randInt63n draws jitter from the same source as the tokens. crypto/rand is
// overkill for a sleep, but it avoids seeding a second generator for the sake
// of a few milliseconds.
func randInt63n(n int64) int64 {
	if n <= 0 {
		return 0
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	v := int64(b[0])<<48 | int64(b[1])<<40 | int64(b[2])<<32 |
		int64(b[3])<<24 | int64(b[4])<<16 | int64(b[5])<<8 | int64(b[6])
	if v < 0 {
		v = -v
	}
	return v % n
}
