package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/battle"
	"github.com/eve-kill/shrike/internal/queue"
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

	// Clear before detecting. Re-detection is the normal case — late killmails
	// keep changing a fight's shape for days — and without this a battle whose
	// boundaries moved would be stored alongside its previous self.
	cleared, err := battle.ClearWindow(ctx, w.Deps.Pool, from, to)
	if err != nil {
		return err
	}
	for _, id := range cleared {
		// Whatever those battles were, they no longer exist under those ids, so
		// any ticker entry pointing at one is now a dead link.
		w.Deps.Ticker.BattleExpired(ctx, id)
	}

	candidates, err := battle.FindCandidates(ctx, w.Deps.Pool, from, to)
	if err != nil {
		return err
	}

	var found int
	for _, c := range candidates {
		kms, atts, err := battle.LoadSystem(ctx, w.Deps.Pool, c.SolarSystemID, from, to)
		if err != nil {
			return err
		}
		// A day can contain several separate fights in one system. Exhaust the
		// refined windows rather than keeping only the first active burst.
		for _, b := range battle.DetectAll(kms, atts) {
			id, err := battle.Store(ctx, w.Deps.Pool, b)
			if err != nil {
				return err
			}
			found++

			// Only the big ones reach the ticker. Every detected battle would be a
			// second killfeed; a hundred and fifty ships is the scale at which the
			// fight is news to people who were not in it.
			if b.KillCount >= battleTickerThreshold {
				w.Deps.Ticker.BattleStarted(ctx, id,
					w.systemName(b.SolarSystemID), w.regionName(b.RegionID),
					b.KillCount, b.IskDestroyed)
			}
		}
	}

	w.Deps.Log.Debug().
		Time("from", from).Time("to", to).
		Int("candidates", len(candidates)).Int("battles", found).
		Msg("battle detection")
	return nil
}

// battleTickerThreshold is how many ships have to die before a battle is worth
// announcing. Detection finds hundreds of small fights a day; the ticker is for
// the ones that matter to people who were not there.
const battleTickerThreshold = 150

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
