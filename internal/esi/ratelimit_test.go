package esi

import (
	"context"
	"sync"
	"testing"
	"time"
)

// testGroup builds a group whose name is unique to the test, so buckets in
// Redis cannot collide between tests running against the same instance.
func testGroup(t *testing.T, limit, window int) Group {
	t.Helper()
	return Group{
		Name:                "test-" + t.Name(),
		PathPrefix:          "test",
		Limit:               limit,
		Window:              window,
		HeaderAuthoritative: true,
	}
}

func TestAcquireConsumesAndDrains(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	r := NewRateLimiter(rdb)
	g := testGroup(t, 6, 60) // three requests at a cost of two

	for i := 1; i <= 3; i++ {
		wait, err := r.Acquire(ctx, g, TokenCost)
		if err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
		if wait != 0 {
			t.Fatalf("acquire %d was refused with a %v wait, but the budget allows 3", i, wait)
		}
	}

	// The fourth must be refused, and the wait must point at the window's end
	// rather than being an arbitrary retry delay.
	wait, err := r.Acquire(ctx, g, TokenCost)
	if err != nil {
		t.Fatal(err)
	}
	if wait <= 0 {
		t.Fatal("a drained bucket allowed a fourth request")
	}
	if wait > time.Duration(g.Window)*time.Second {
		t.Errorf("wait of %v exceeds the %ds window", wait, g.Window)
	}

	state, err := r.Peek(ctx, g)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Seeded {
		t.Error("a used bucket should report as seeded")
	}
	if state.Remaining != 0 {
		t.Errorf("remaining = %d, want 0", state.Remaining)
	}
}

// The whole reason the bucket lives in Redis rather than in each process: no
// number of concurrent callers may spend more than the budget. If this test
// ever fails, the deployment is bursting ESI and will get 420'd.
func TestAcquireIsAtomicUnderConcurrency(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	r := NewRateLimiter(rdb)

	const limit = 40
	g := testGroup(t, limit, 60)
	wantGrants := limit / TokenCost

	const callers = 200
	var granted counter
	var wg sync.WaitGroup
	start := make(chan struct{})

	for range callers {
		wg.Go(func() {
			// Released together, so the acquires genuinely contend rather than
			// arriving in a queue.
			<-start
			wait, err := r.Acquire(ctx, g, TokenCost)
			if err == nil && wait == 0 {
				granted.inc()
			}
		})
	}
	close(start)
	wg.Wait()

	if got := granted.get(); got != wantGrants {
		t.Errorf("%d of %d callers were granted tokens, want exactly %d — the bucket oversubscribed by %d requests",
			got, callers, wantGrants, got-wantGrants)
	}
}

// A fixed window refills in one step at the boundary, which is what ESI does.
func TestBucketRefillsAfterWindow(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	r := NewRateLimiter(rdb)
	g := testGroup(t, 2, 1) // one request per second

	if wait, err := r.Acquire(ctx, g, TokenCost); err != nil || wait != 0 {
		t.Fatalf("first acquire: wait=%v err=%v", wait, err)
	}
	if wait, _ := r.Acquire(ctx, g, TokenCost); wait == 0 {
		t.Fatal("second acquire inside the window should have been refused")
	}

	time.Sleep(1100 * time.Millisecond)

	if wait, err := r.Acquire(ctx, g, TokenCost); err != nil || wait != 0 {
		t.Errorf("after the window rolled over: wait=%v err=%v", wait, err)
	}
}

// ESI's own accounting wins when the window has rolled over on its side, but a
// *higher* remaining inside the current window must be ignored: our count
// includes requests ESI had not yet seen, and adopting the larger number would
// spend them twice.
func TestApplyHeadersMergeRules(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	r := NewRateLimiter(rdb)
	g := testGroup(t, 100, 60)

	if _, err := r.Acquire(ctx, g, TokenCost); err != nil {
		t.Fatal(err)
	}

	// Lower than ours: adopt it, we were behind.
	if err := r.ApplyHeaders(ctx, g, 50, 55*time.Second); err != nil {
		t.Fatal(err)
	}
	if state, _ := r.Peek(ctx, g); state.Remaining != 50 {
		t.Errorf("a lower header remaining was not adopted: %d", state.Remaining)
	}

	// Higher, same window: ignore.
	if err := r.ApplyHeaders(ctx, g, 90, 50*time.Second); err != nil {
		t.Fatal(err)
	}
	if state, _ := r.Peek(ctx, g); state.Remaining != 50 {
		t.Errorf("a higher header remaining inside the window was adopted: %d", state.Remaining)
	}

	// A reset comfortably in the future means ESI rolled over; trust it whole.
	if err := r.ApplyHeaders(ctx, g, 300, 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	state, _ := r.Peek(ctx, g)
	if state.Remaining != 300 {
		t.Errorf("a newer window was not adopted wholesale: %d", state.Remaining)
	}
	if time.Until(state.ResetAt) < 9*time.Minute {
		t.Errorf("reset_at was not moved forward: %v", state.ResetAt)
	}
}

// /killmails/{id}/{hash}/ reports remaining with no reset. Without this path the
// bucket drains to zero with no feedback and blocks every killmail until the
// window rolls over on its own.
func TestApplyRemainingWithoutReset(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	r := NewRateLimiter(rdb)
	g := testGroup(t, 100, 60)

	if _, err := r.Acquire(ctx, g, TokenCost); err != nil {
		t.Fatal(err)
	}
	before, _ := r.Peek(ctx, g)

	if err := r.ApplyRemaining(ctx, g, 3333); err != nil {
		t.Fatal(err)
	}

	after, _ := r.Peek(ctx, g)
	if after.Remaining != 3333 {
		t.Errorf("remaining = %d, want 3333", after.Remaining)
	}
	// reset_at must survive untouched: it is the only thing that will refill
	// the bucket, and moving it would either stall or over-grant.
	if !after.ResetAt.Equal(before.ResetAt) {
		t.Errorf("reset_at moved from %v to %v", before.ResetAt, after.ResetAt)
	}
}

// The hand-built presets are the only ground truth for endpoints that emit no
// headers. Anything that did arrive on such a response must be ignored.
func TestPresetGroupsIgnoreHeaders(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	ctx := context.Background()
	r := NewRateLimiter(rdb)

	g := testGroup(t, 60, 60)
	g.HeaderAuthoritative = false

	if _, err := r.Acquire(ctx, g, TokenCost); err != nil {
		t.Fatal(err)
	}
	before, _ := r.Peek(ctx, g)

	if err := r.ApplyHeaders(ctx, g, 1, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyRemaining(ctx, g, 1); err != nil {
		t.Fatal(err)
	}

	after, _ := r.Peek(ctx, g)
	if after.Remaining != before.Remaining {
		t.Errorf("a preset-only group adopted header state: %d → %d", before.Remaining, after.Remaining)
	}
}

// An untouched bucket reads as full, not empty. The opposite would make every
// group unusable until something happened to seed it.
func TestPeekUnseededBucketReadsFull(t *testing.T) {
	rdb := testRedis(t)
	clearESIState(t, rdb)
	t.Cleanup(func() { clearESIState(t, rdb) })

	r := NewRateLimiter(rdb)
	g := testGroup(t, 123, 60)

	state, err := r.Peek(context.Background(), g)
	if err != nil {
		t.Fatal(err)
	}
	if state.Seeded {
		t.Error("an untouched bucket reported as seeded")
	}
	if state.Remaining != 123 {
		t.Errorf("remaining = %d, want the full limit of 123", state.Remaining)
	}
}
