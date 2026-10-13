// Package requeststats measures work attached to one development request.
// It retains counters and durations only, never SQL, arguments or user data.
package requeststats

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type contextKey struct{}

type Stats struct {
	mu       sync.Mutex
	started  time.Time
	snapshot Snapshot
}

type Snapshot struct {
	Server, Load, Render, Database, PoolWait time.Duration
	Queries, Errors, Batches, Copies         int
}

func Start(ctx context.Context) (context.Context, *Stats) {
	stats := &Stats{started: time.Now()}
	return context.WithValue(ctx, contextKey{}, stats), stats
}

func From(ctx context.Context) *Stats {
	stats, _ := ctx.Value(contextKey{}).(*Stats)
	return stats
}

func (s *Stats) AddLoad(d time.Duration) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.snapshot.Load += d
	s.mu.Unlock()
}
func (s *Stats) AddRender(d time.Duration) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.snapshot.Render += d
	s.mu.Unlock()
}
func (s *Stats) AddQuery() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.snapshot.Queries++
	s.mu.Unlock()
}
func (s *Stats) AddBatch() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.snapshot.Batches++
	s.mu.Unlock()
}
func (s *Stats) AddCopy() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.snapshot.Copies++
	s.mu.Unlock()
}
func (s *Stats) AddDatabase(d time.Duration, failed bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.Database += d
	if failed {
		s.snapshot.Errors++
	}
}
func (s *Stats) AddPoolWait(d time.Duration) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.snapshot.PoolWait += d
	s.mu.Unlock()
}
func (s *Stats) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := s.snapshot
	snapshot.Server = time.Since(s.started)
	return snapshot
}

// ServerTiming uses durations only for time metrics. Counts use descriptions,
// so DevTools doesn't present a query count as a fictional duration.
func (s Snapshot) ServerTiming() string {
	ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
	return fmt.Sprintf(`app;dur=%.3f;desc="Server", load;dur=%.3f;desc="Data loading", render;dur=%.3f;desc="Templates", db;dur=%.3f;desc="Database client time", dbwait;dur=%.3f;desc="Pool acquisition", dbcount;desc="%d", dberrors;desc="%d", dbbatches;desc="%d", dbcopies;desc="%d"`,
		ms(s.Server), ms(s.Load), ms(s.Render), ms(s.Database), ms(s.PoolWait), s.Queries, s.Errors, s.Batches, s.Copies)
}
