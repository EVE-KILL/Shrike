package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/riverqueue/river"
)

const killCategoryRefreshInterval = time.Hour

type KillCategoryTopWorker struct {
	river.WorkerDefaults[queue.KillCategoryTopArgs]
	Deps *Deps
}

func (w *KillCategoryTopWorker) Timeout(*river.Job[queue.KillCategoryTopArgs]) time.Duration {
	return 45 * time.Second
}

func (w *KillCategoryTopWorker) Work(ctx context.Context, job *river.Job[queue.KillCategoryTopArgs]) error {
	kind := job.Args.Category
	if _, known := killtype.Predicates()[kind]; !known {
		return river.JobCancel(fmt.Errorf("unknown kill category %q", kind))
	}
	var fresh bool
	err := w.Deps.Pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM kill_category_top_snapshots WHERE kind = $1 AND prepared_at > $2)`,
		kind, time.Now().Add(-killCategoryRefreshInterval)).Scan(&fresh)
	if err != nil || fresh {
		return err
	}
	if err := api.PrepareKillCategoryTop(ctx, w.Deps.Pool, w.Deps.Redis, kind); err != nil {
		return fmt.Errorf("refresh %s leaderboard: %w", kind, err)
	}
	return nil
}

func (d *Deps) cronKillCategoryTop(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("kill_category_top")
	}
	rows, err := d.Pool.Query(ctx, `
		SELECT categories.kind, snapshot.prepared_at
		FROM unnest($1::text[]) WITH ORDINALITY AS categories(kind, position)
		LEFT JOIN kill_category_top_snapshots snapshot USING (kind)
		ORDER BY snapshot.prepared_at NULLS FIRST, categories.position`, killtype.Types)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	now := time.Now()
	var due []river.JobArgs
	var missing int
	var oldest time.Duration
	for rows.Next() {
		var kind string
		var preparedAt *time.Time
		if err := rows.Scan(&kind, &preparedAt); err != nil {
			return "", err
		}
		if preparedAt == nil {
			missing++
			due = append(due, queue.KillCategoryTopArgs{Category: kind})
			continue
		}
		age := now.Sub(*preparedAt)
		oldest = max(oldest, age)
		if age >= killCategoryRefreshInterval {
			due = append(due, queue.KillCategoryTopArgs{Category: kind})
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	telemetry.Gauge(ctx, "leaderboard.snapshots_missing", float64(missing))
	telemetry.Gauge(ctx, "leaderboard.oldest_snapshot_age_seconds", oldest.Seconds())
	queued, err := queue.DispatchMany(ctx, d.Queue, due, queue.Live)
	return fmt.Sprintf("%d/%d leaderboard snapshots ready, oldest %s, %d due, %d queued",
		len(killtype.Types)-missing, len(killtype.Types), oldest.Round(time.Second), len(due), queued), err
}
