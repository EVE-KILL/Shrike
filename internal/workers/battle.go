package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/battle"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// BattleDetectionWorker finds the battles in one time window.
type BattleDetectionWorker struct {
	river.WorkerDefaults[queue.BattleDetectionArgs]
	Deps *Deps
}

func (w *BattleDetectionWorker) Work(ctx context.Context, job *river.Job[queue.BattleDetectionArgs]) error {
	from, to := job.Args.From, job.Args.To
	if !to.After(from) {
		return river.JobCancel(fmt.Errorf("window %s..%s is empty", from, to))
	}

	systemID := job.Args.SystemID
	var dirtyAt time.Time
	if systemID > 0 {
		err := w.Deps.Pool.QueryRow(ctx, `
			SELECT dirtied_at FROM battle_dirty_days WHERE solar_system_id = $1 AND activity_date = $2::date`,
			systemID, from.Format(time.DateOnly)).Scan(&dirtyAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	candidates, err := battle.FindHotspotWindows(
		ctx,
		w.Deps.Pool,
		from,
		to,
		battleHotspotMinimum,
		systemID,
	)
	if err != nil {
		return err
	}

	var found int
	for _, c := range candidates {
		// The hourly candidate query deliberately stays cheap. Extend its
		// boundaries before doing the detailed five-minute segmentation so a
		// fight beginning just before a hot hour or ending just after it is not
		// clipped.
		start, end, err := battle.ExtendWindow(ctx, w.Deps.Pool, c.SolarSystemID, c.Start.Add(-battleEdgeExtension), c.End.Add(battleEdgeExtension))
		if err != nil {
			return err
		}
		kms, atts, err := battle.LoadSystem(
			ctx,
			w.Deps.Pool,
			c.SolarSystemID,
			start, end,
		)
		if err != nil {
			return err
		}
		// A day can contain several separate fights in one system. Exhaust the
		// refined windows rather than keeping only the first active burst.
		for _, b := range battle.DetectAll(kms, atts) {
			id, err := battle.Store(ctx, w.Deps.Pool, b, w.Deps.Queue)
			if err != nil {
				return err
			}
			found++

			// Only the big ones reach the ticker. Every detected battle would be a
			// second killfeed; a hundred and fifty ships is the scale at which the
			// fight is news to people who were not in it.
			if b.KillCount >= battleTickerThreshold && time.Since(b.End) < battleRecentWindow && w.Deps.Ticker != nil {
				w.Deps.Ticker.BattleStarted(ctx, id,
					w.systemName(b.SolarSystemID), w.regionName(b.RegionID),
					b.KillCount, b.IskDestroyed)
			}
		}
	}

	if systemID > 0 {
		if err := w.refreshBattleReports(ctx, systemID, from, to, dirtyAt); err != nil {
			return err
		}
	}

	w.Deps.Log.Debug().
		Time("from", from).Time("to", to).
		Int("candidates", len(candidates)).Int("battles", found).
		Msg("battle detection")
	return nil
}

func (w *BattleDetectionWorker) refreshBattleReports(ctx context.Context, systemID int32, from, to, dirtyAt time.Time) error {
	return pgx.BeginFunc(ctx, w.Deps.Pool, func(tx pgx.Tx) error {
		// A late kill can change the report without changing the detector's totals.
		rows, err := tx.Query(ctx, `
			UPDATE battles SET updated_at = clock_timestamp(), report_pending = true
			WHERE solar_system_id = $1 AND start_time < $3 AND end_time >= $2
			RETURNING battle_id`, systemID, from, to)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return err
		}
		if err := queue.EnqueueBattleReportsTx(ctx, tx, w.Deps.Queue, ids); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			DELETE FROM battle_dirty_days WHERE solar_system_id = $1 AND activity_date = $2::date
			  AND dirtied_at = $3`, systemID, from.Format(time.DateOnly), dirtyAt)
		return err
	})
}

func (d *Deps) cronDirtyBattles(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("battle_dirty_refresh")
	}
	rows, err := d.Pool.Query(ctx, `
		SELECT solar_system_id, activity_date FROM battle_dirty_days
		ORDER BY dirtied_at LIMIT 40`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var args []river.JobArgs
	for rows.Next() {
		var systemID int32
		var day time.Time
		if err := rows.Scan(&systemID, &day); err != nil {
			return "", err
		}
		args = append(args, queue.BattleDetectionArgs{From: day, To: day.AddDate(0, 0, 1), SystemID: systemID})
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	n, err := queue.DispatchMany(ctx, d.Queue, args, queue.RecentBackfill)
	return fmt.Sprintf("%d dirty battle windows queued", n), err
}

type BattleReportWorker struct {
	river.WorkerDefaults[queue.BattleReportArgs]
	Deps *Deps
}

func (w *BattleReportWorker) Work(ctx context.Context, job *river.Job[queue.BattleReportArgs]) error {
	first, last := job.Args.FirstID, job.Args.LastID
	if first < 1 || last < first || last-first >= queue.BattleReportBatchSize {
		return river.JobCancel(fmt.Errorf("invalid battle report range %d..%d", first, last))
	}
	_, err := api.PrepareBattleReports(ctx, w.Deps.Pool, first, last)
	return err
}

// battleTickerThreshold is how many ships have to die before a battle is worth
// announcing. Detection finds hundreds of small fights a day; the ticker is for
// the ones that matter to people who were not there.
const battleTickerThreshold = 150

const (
	// These are the backend detector's phase-one parameters. Five kills in a
	// five-minute segment makes the final cut; ten kills in an hour merely
	// decides whether the more expensive attacker load is worth doing.
	battleHotspotMinimum = 10
	battleEdgeExtension  = 30 * time.Minute
)

// systemName resolves a system for display, tolerating an unknown id.
func (w *BattleDetectionWorker) systemName(id int32) string {
	if w.Deps.Cache == nil {
		return "Unknown"
	}
	if s, ok := w.Deps.Cache.System(id); ok && s.Name != "" {
		return s.Name
	}
	return "Unknown"
}

// regionName resolves a region for display. Empty rather than "Unknown" — the
// announcement omits the region entirely when it cannot be named.
func (w *BattleDetectionWorker) regionName(id int32) string {
	if w.Deps.Cache == nil || id == 0 {
		return ""
	}
	if r, ok := w.Deps.Cache.Region(id); ok {
		return r.Name
	}
	return ""
}

// Detection windows.
const (
	// battleRecentWindow is re-scanned every hour. Six hours rather than one,
	// because a battle that started five hours ago and is still going has to be
	// re-detected with its full extent — a window of one hour would find only
	// its tail and store a fight that looks far smaller than it was.
	battleRecentWindow = 6 * time.Hour

	// battleBackfillDays is re-scanned once a day. Late-arriving killmails keep
	// changing a fight's shape for a while, so recent days are re-detected
	// until they settle.
	battleBackfillDays = 14
)

// cronBattleDetection queues the detection windows.
func (d *Deps) cronBattleDetection(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("battle_detection")
	}

	now := time.Now().UTC()
	args := []river.JobArgs{
		queue.BattleDetectionArgs{From: now.Add(-battleRecentWindow), To: now},
	}

	// The daily backfill runs once, at the top of the UTC day. Queuing it every
	// hour would re-scan a fortnight of killmails twenty-four times a day for
	// results that stop changing after the first.
	if now.Hour() == 0 {
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		for day := 1; day <= battleBackfillDays; day++ {
			start := midnight.AddDate(0, 0, -day)
			args = append(args, queue.BattleDetectionArgs{
				From: start, To: start.AddDate(0, 0, 1),
			})
		}
	}

	n, err := queue.DispatchMany(ctx, d.Queue, args, queue.RecentBackfill)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d detection windows queued", n), nil
}
