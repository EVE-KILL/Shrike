package workers

import (
	"context"
	"errors"

	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/stats"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// StatsWriterWorker folds one killmail into the aggregate counters.
type StatsWriterWorker struct {
	river.WorkerDefaults[queue.StatsWriterArgs]
	Deps *Deps
}

func (w *StatsWriterWorker) Work(ctx context.Context, job *river.Job[queue.StatsWriterArgs]) error {
	km, attackers, err := stats.Load(ctx, w.Deps.Pool, job.Args.KillmailID)
	if errors.Is(err, stats.ErrNotStored) {
		// The killmail was deleted between dispatch and now. Nothing to count,
		// and no number of retries will bring it back.
		return nil
	}
	if err != nil {
		return err
	}

	// The counters are additive, so the writes and the completion bit have to
	// share one transaction. River uniqueness prevents two pending jobs, but it
	// does not prevent retrying a job whose first attempt committed its counters
	// and lost the acknowledgement.
	a := stats.NewAccumulator()
	a.Add(km, attackers)

	_, err = killmail.RunDBEffect(
		ctx,
		w.Deps.Pool,
		job.Args.KillmailID,
		killmail.EffectStatsWritten,
		func(ctx context.Context, tx pgx.Tx) (bool, error) {
			_, err := stats.WritePeriodTx(
				ctx, tx, a, km.KillmailTime, stats.PeriodDaily, true, true,
			)
			return true, err
		},
		killmail.EffectOptions{},
	)
	return err
}
