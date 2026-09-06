package workers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/intelrollup"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
)

func TestHistoricalRollupBatchKeepsOriginalDeadline(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("requires explicit disposable TEST_DATABASE_URL")
	}
	ctx := context.Background()
	pool := testPool(t)
	client, err := queue.New(queue.Options{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	args := queue.CharacterIntelRollupArgs{Day: day.Format("2006-01-02")}
	before := time.Now()
	delay := intelrollup.RefreshDelay(day, time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC))
	first, err := queue.DispatchAt(ctx, client, args, queue.DormantBackfill, delay)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM river_job WHERE id=$1", first.Job.ID) })
	if first.Job.ScheduledAt.Before(before.Add(6*time.Hour)) || first.Job.ScheduledAt.After(time.Now().Add(6*time.Hour)) {
		t.Fatalf("historical deadline = %v", first.Job.ScheduledAt)
	}
	// A further late kill must neither create a second job nor indefinitely
	// extend the batching window. Maintenance must not pull it forward either.
	second, err := queue.DispatchAt(ctx, client, args, queue.RecentBackfill, 12*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate || second.Job.ID != first.Job.ID {
		t.Fatal("duplicate historical rebuild queued")
	}
	if _, err := queue.DispatchMany(ctx, client, []river.JobArgs{args}, queue.DormantBackfill); err != nil {
		t.Fatal(err)
	}
	var deadline time.Time
	if err := pool.QueryRow(ctx, "SELECT scheduled_at FROM river_job WHERE id=$1", first.Job.ID).Scan(&deadline); err != nil {
		t.Fatal(err)
	}
	if !deadline.Equal(first.Job.ScheduledAt) {
		t.Fatalf("deadline moved: %v -> %v", first.Job.ScheduledAt, deadline)
	}
}
