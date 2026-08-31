package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/intelrollup"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
)

type CharacterIntelRollupWorker struct {
	river.WorkerDefaults[queue.CharacterIntelRollupArgs]
	Deps *Deps
}

func (w *CharacterIntelRollupWorker) Work(ctx context.Context, job *river.Job[queue.CharacterIntelRollupArgs]) error {
	day, err := time.Parse("2006-01-02", job.Args.Day)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid intel rollup day %q: %w", job.Args.Day, err))
	}
	result, rebuilt, err := intelrollup.ProcessDirtyDay(ctx, w.Deps.Pool, day)
	if err != nil || !rebuilt {
		return err
	}
	w.Deps.Log.Info().Str("component", "character_intel_rollup").Str("day", job.Args.Day).
		Int64("characters", result.Characters).Int64("ships", result.Ships).
		Int64("targets", result.Targets).Msg("character intel day rebuilt")
	dirty, err := intelrollup.IsDirty(ctx, w.Deps.Pool, day)
	if err != nil {
		return err
	}
	if dirty {
		return river.JobSnooze(time.Minute)
	}
	return nil
}
