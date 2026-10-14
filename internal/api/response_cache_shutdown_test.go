package api

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func waitCacheSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatal("cache lifecycle signal timed out")
	}
}

func TestResponseCacheCloseCancelsRefreshAndWaitsForCleanup(t *testing.T) {
	synctest.Test(t, testResponseCacheCloseCancelsRefreshAndWaitsForCleanup)
}

func testResponseCacheCloseCancelsRefreshAndWaitsForCleanup(t *testing.T) {
	cache := newResponseCache(nil, 1024)
	started, cancelled, release, closed := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer func() { unblock(); cache.Close() }()
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	cancelRequest()
	var healthyAtStart atomic.Bool
	cache.RefreshContext(requestCtx, "refresh", func(ctx context.Context) (any, error) {
		healthyAtStart.Store(ctx.Err() == nil)
		close(started)
		<-ctx.Done()
		close(cancelled)
		<-release // Simulate cleanup that still needs live dependencies.
		return nil, ctx.Err()
	})
	waitCacheSignal(t, started)
	if !healthyAtStart.Load() {
		t.Fatal("client cancellation cancelled a detached refresh")
	}
	go func() { (&Service{cache: cache}).Close(); close(closed) }()
	waitCacheSignal(t, cancelled)
	synctest.Wait()
	select {
	case <-closed:
		t.Fatal("service closed before refresh cleanup finished")
	default:
	}
	unblock()
	waitCacheSignal(t, closed)
	var called atomic.Bool
	_, err, _ := cache.LoadOnce(context.Background(), "new", func() (any, error) {
		called.Store(true)
		return nil, nil
	})
	if !errors.Is(err, context.Canceled) || called.Load() {
		t.Fatalf("loader ran after close: called=%v err=%v", called.Load(), err)
	}
}

func TestResponseCacheCloseDrainsLoadAfterItsHTTPWaiterCancels(t *testing.T) {
	synctest.Test(t, testResponseCacheCloseDrainsLoadAfterItsHTTPWaiterCancels)
}

func testResponseCacheCloseDrainsLoadAfterItsHTTPWaiterCancels(t *testing.T) {
	cache := newResponseCache(nil, 1024)
	started, release, returned, closed := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer func() { unblock(); cache.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var waiterErr error
	go func() {
		_, waiterErr, _ = cache.LoadOnce(ctx, "miss", func() (any, error) {
			close(started)
			<-release
			return "loaded", nil
		})
		close(returned)
	}()
	waitCacheSignal(t, started)
	cancel()
	waitCacheSignal(t, returned)
	if !errors.Is(waiterErr, context.Canceled) {
		t.Fatalf("HTTP waiter returned %v", waiterErr)
	}
	go func() { cache.Close(); close(closed) }()
	synctest.Wait()
	select {
	case <-closed:
		t.Fatal("cache closed before the abandoned loader finished")
	default:
	}
	unblock()
	waitCacheSignal(t, closed)
}
