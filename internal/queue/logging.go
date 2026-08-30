package queue

import (
	"context"
	"errors"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog"
)

// jobLogMiddleware reports the lifecycle of dispatched jobs.
//
// Cron jobs report through cron.Worker instead. They have useful domain
// details such as the cron name, report, and skip reason that a generic River
// middleware cannot see, and logging them here as well would duplicate every
// scheduled run.
type jobLogMiddleware struct {
	river.MiddlewareDefaults

	log zerolog.Logger
}

func newJobLogMiddleware(logger zerolog.Logger) *jobLogMiddleware {
	return &jobLogMiddleware{log: logger}
}

func (m *jobLogMiddleware) Work(
	ctx context.Context,
	job *rivertype.JobRow,
	doInner func(context.Context) error,
) error {
	if job.Queue == CronQueue {
		return doInner(ctx)
	}

	logger := m.log.With().
		Int64("job_id", job.ID).
		Str("queue", job.Queue).
		Str("kind", job.Kind).
		Int("attempt", job.Attempt).
		Int("max_attempts", job.MaxAttempts).
		Logger()

	logger.Info().Msg("job started")
	started := time.Now()
	err := doInner(ctx)
	elapsed := time.Since(started)

	if err == nil {
		logger.Info().Dur("duration", elapsed).Msg("job completed")
		return nil
	}

	if snoozeErr, ok := errors.AsType[*river.JobSnoozeError](err); ok {
		logger.Info().
			Dur("duration", elapsed).
			Dur("snooze_for", snoozeErr.Duration).
			Msg("job snoozed")
		return err
	}

	if _, ok := errors.AsType[*river.JobCancelError](err); ok {
		logger.Warn().
			Err(err).
			Dur("duration", elapsed).
			Msg("job cancelled")
		return err
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, river.ErrJobCancelledRemotely) {
		logger.Warn().
			Err(err).
			Dur("duration", elapsed).
			Msg("job interrupted")
		return err
	}

	if job.Attempt < job.MaxAttempts {
		logger.Warn().
			Err(err).
			Dur("duration", elapsed).
			Msg("job failed; retry scheduled")
		return err
	}

	logger.Error().
		Err(err).
		Dur("duration", elapsed).
		Msg("job failed permanently")
	return err
}

var _ rivertype.WorkerMiddleware = (*jobLogMiddleware)(nil)
