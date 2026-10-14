package zkb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestListenerRetriesClientTimeoutWithoutAdvancingCursor(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		first := len(paths) == 1
		mu.Unlock()
		if first {
			// Real Client.Timeout expires while the listener's context stays live.
			<-r.Context().Done()
			return
		}
		_, _ = io.WriteString(w, `{"killmail_id":111,"hash":"fixture","sequence_id":101}`)
	}))
	defer server.Close()
	client := server.Client()
	client.Timeout = 100 * time.Millisecond
	store := newFakeStore(100)
	listener := &Listener{Client: &Client{BaseURL: server.URL, HTTP: client}, Store: store}
	guard, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	ctx, cancel := context.WithCancel(guard)
	defer cancel()
	pauses := 0
	listener.Sleep = func(ctx context.Context, wait time.Duration) error {
		pauses++
		if wait != WaitOnCaughtUp || listener.Stats().Sequence != 100 || ctx.Err() != nil {
			t.Errorf("timeout retry: wait=%v cursor=%d context=%v", wait, listener.Stats().Sequence, ctx.Err())
		}
		return ctx.Err()
	}
	listener.OnEvent = func(event Event) {
		switch event.Kind {
		case "error":
			if !errors.Is(event.Err, context.DeadlineExceeded) || ctx.Err() != nil {
				t.Errorf("expected independent request deadline, got %v (parent=%v)", event.Err, ctx.Err())
			}
		case "new":
			cancel()
		}
	}
	stats, err := listener.Start(ctx)
	if !errors.Is(err, context.Canceled) || stats.Accepted != 1 || stats.Errors != 1 || stats.Sequence != 101 || pauses != 1 {
		t.Fatalf("listener failed to recover: stats=%+v pauses=%d err=%v", stats, pauses, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 2 || paths[0] != "/101.json" || paths[1] != "/101.json" {
		t.Fatalf("did not retry the same sequence: %v", paths)
	}
	if ids := store.acceptedIDs(); len(ids) != 1 || ids[0] != 111 {
		t.Fatalf("accepted killmails = %v", ids)
	}
	if saves := store.saves(); len(saves) != 1 || saves[0] != 101 {
		t.Fatalf("saved cursor = %v", saves)
	}
}

func TestListenerStopsForParentCancellationDuringRequest(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "canceled", true: "deadline"}[deadline], func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			want := context.Canceled
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 100*time.Millisecond)
				want = context.DeadlineExceeded
			}
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				if !deadline {
					cancel()
				}
				<-r.Context().Done()
			}))
			defer server.Close()
			store := newFakeStore(100)
			listener := &Listener{Client: &Client{BaseURL: server.URL, HTTP: server.Client()}, Store: store}
			listener.OnEvent = func(event Event) { t.Errorf("shutdown emitted retry event: %+v", event) }
			listener.Sleep = func(context.Context, time.Duration) error { t.Error("shutdown retried"); return want }
			stats, err := listener.Start(ctx)
			if !errors.Is(err, want) || stats.Errors != 0 || stats.Sequence != 100 {
				t.Fatalf("shutdown = %+v, %v", stats, err)
			}
			if saves := store.saves(); len(saves) != 1 || saves[0] != 100 {
				t.Fatalf("shutdown cursor = %v", saves)
			}
		})
	}
}
