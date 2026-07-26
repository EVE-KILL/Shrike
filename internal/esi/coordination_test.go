package esi

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestClaimIsExclusive(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	c := NewCoordination(rdb)
	url := "https://esi.test/" + t.Name()

	first, err := c.TryClaim(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if first == "" {
		t.Fatal("the first claim on an idle URL was refused")
	}

	second, err := c.TryClaim(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if second != "" {
		t.Fatal("a second worker claimed a URL already in flight")
	}

	c.ReleaseClaim(ctx, url, first)

	third, err := c.TryClaim(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if third == "" {
		t.Fatal("the claim was not released")
	}
}

// Release is fenced so that a worker whose claim expired mid-request cannot
// delete the claim of whoever took over. Without the fence, a slow request
// would repeatedly free a lock it no longer holds and singleflight would stop
// deduplicating exactly when it matters.
func TestReleaseIsFencedByToken(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	c := NewCoordination(rdb)
	url := "https://esi.test/" + t.Name()

	owner, err := c.TryClaim(ctx, url)
	if err != nil {
		t.Fatal(err)
	}

	// Somebody else's token must not free this claim.
	c.ReleaseClaim(ctx, url, "0000000000000000000000000000dead")

	if again, _ := c.TryClaim(ctx, url); again != "" {
		t.Fatal("a foreign token released the claim")
	}

	c.ReleaseClaim(ctx, url, owner)
	if again, _ := c.TryClaim(ctx, url); again == "" {
		t.Fatal("the rightful owner could not release the claim")
	}
}

func TestWaitForClaimReturnsWhenCleared(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	c := NewCoordination(rdb)
	url := "https://esi.test/" + t.Name()

	token, err := c.TryClaim(ctx, url)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(150 * time.Millisecond)
		c.ReleaseClaim(ctx, url, token)
	}()

	start := time.Now()
	if !c.WaitForClaim(ctx, url) {
		t.Fatal("the wait timed out even though the claim was released")
	}
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("returned after %v — it cannot have observed the release", elapsed)
	}
	if elapsed > 5*time.Second {
		t.Errorf("took %v to notice a release 150ms in", elapsed)
	}
}

// A cancelled context must stop the wait, or a shutdown hangs for the full
// 30-second claim window.
func TestWaitForClaimHonoursCancellation(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	c := NewCoordination(rdb)
	url := "https://esi.test/" + t.Name()

	if _, err := c.TryClaim(context.Background(), url); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	c.WaitForClaim(ctx, url)
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("ignored cancellation for %v", elapsed)
	}
}

// The sequential lock exists for endpoints that 420 under a concurrent burst
// even with tokens in hand. Its only job is that exactly one holder exists at a
// time — so that is what is asserted, by having every holder check.
func TestSequentialLockAdmitsOneAtATime(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	c := NewCoordination(rdb)
	group := "test-" + t.Name()

	var inside atomic.Int32
	var overlaps atomic.Int32
	var acquired counter

	const workers = 12
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			token, err := c.AcquireSequential(ctx, group)
			if err != nil || token == "" {
				return
			}
			acquired.inc()

			if inside.Add(1) != 1 {
				overlaps.Add(1)
			}
			// Long enough that a broken lock would reliably overlap.
			time.Sleep(30 * time.Millisecond)
			inside.Add(-1)

			c.ReleaseSequential(ctx, group, token)
		}()
	}
	wg.Wait()

	if overlaps.Load() != 0 {
		t.Errorf("%d workers held the sequential lock at the same time", overlaps.Load())
	}
	if acquired.get() != workers {
		t.Errorf("only %d of %d workers ever got the lock", acquired.get(), workers)
	}
}

// Two URLs must not share a claim. They are keyed by hash, so a collision here
// would silently serialise unrelated requests.
func TestClaimsArePerURL(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	c := NewCoordination(rdb)

	a, err := c.TryClaim(ctx, "https://esi.test/a")
	if err != nil || a == "" {
		t.Fatalf("claim a: %v", err)
	}
	b, err := c.TryClaim(ctx, "https://esi.test/b")
	if err != nil || b == "" {
		t.Fatalf("claim b was blocked by an unrelated URL: %v", err)
	}
}
