package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/retry"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"

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
) (result error) {
	name := job.Kind
	attrs := []string{"queue", job.Queue, "kind", job.Kind}
	if job.Queue == CronQueue {
		var args CronArgs
		if json.Unmarshal(job.EncodedArgs, &args) == nil && jobs.CronByName(args.Name) != nil {
			name = args.Name
		} else {
			name = "unknown"
		}
		attrs = append(attrs, "cron", name)
	}
	var metadata struct {
		Sentry telemetry.TraceCarrier `json:"_sentry"`
	}
	if telemetry.Enabled() {
		_ = json.Unmarshal(job.Metadata, &metadata)
	}
	ctx, task := telemetry.StartTaskWithTrace(ctx, "queue.job", name, metadata.Sentry, attrs...)
	if task != nil {
		sentry.GetHubFromContext(ctx).Scope().SetContext("job", sentry.Context{"id": job.ID, "attempt": job.Attempt, "max_attempts": job.MaxAttempts})
		if !job.ScheduledAt.IsZero() {
			telemetry.Duration(ctx, "queue.wait", max(0, time.Since(job.ScheduledAt)), telemetry.Attributes("queue", job.Queue))
		}
		defer func() {
			if value := recover(); value != nil {
				sentry.GetHubFromContext(ctx).RecoverWithContext(ctx, value)
				task.End("panic", fmt.Errorf("worker panic"))
				panic(value)
			}
			outcome := "completed"
			switch {
			case result == nil:
			case errors.Is(result, context.Canceled), errors.Is(result, river.ErrJobCancelledRemotely):
				outcome = "interrupted"
			default:
				if _, ok := errors.AsType[*river.JobSnoozeError](result); ok {
					outcome = "snoozed"
				} else if _, ok := errors.AsType[*river.JobCancelError](result); ok {
					outcome = "cancelled"
				} else {
					outcome = "retry"
					if job.Attempt >= job.MaxAttempts {
						outcome = "failed"
					}
					// Recoverable upstream attempts remain in warning logs, traces
					// and retry metrics. Exhaustion and unexpected failures still
					// become error issues immediately.
					if outcome == "failed" || ctx.Err() != nil || !retry.Transient(result) {
						telemetry.Capture(ctx, result)
					}
				}
			}
			observedErr := result
			if outcome == "snoozed" || outcome == "cancelled" {
				observedErr = nil
			}
			task.End(outcome, observedErr)
		}()
	}
	if job.Queue == CronQueue {
		return runWithDeferral(ctx, doInner)
	}

	logger := m.log.With().
		Ctx(ctx).Bool("sentry_handled", true).
		Int64("job_id", job.ID).
		Str("queue", job.Queue).
		Str("kind", job.Kind).
		Int("attempt", job.Attempt).
		Int("max_attempts", job.MaxAttempts).
		Logger()

	logger.Info().Msg("job started")
	started := time.Now()
	err := runWithDeferral(ctx, doInner)
	elapsed := time.Since(started)

	if err == nil {
		// River persists this under metadata.output when the job is finalized.
		// Queue handlers don't currently return domain-specific values, so keep
		// a small durable outcome alongside the existing args/timing instead of
		// leaving successful jobs with log-only evidence.
		_ = river.RecordOutput(ctx, map[string]any{
			"status":      "completed",
			"duration_ms": elapsed.Milliseconds(),
		})
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

func runWithDeferral(ctx context.Context, run func(context.Context) error) error {
	err := run(ctx)
	if deferred, ok := retry.Deferral(err); ok {
		delay := max(time.Second, deferred.After)
		// Jitter only adds time; a server's Retry-After remains the minimum.
		jitter := time.Duration(rand.Int64N(int64(min(time.Second, delay/10)))) //nolint:gosec // Queue pacing, not a security decision.
		jitter = min(jitter, time.Duration(math.MaxInt64)-delay)
		telemetry.Count(ctx, "queue.deferrals", 1, telemetry.Attributes("operation", deferred.Operation, "reason", deferred.Reason))
		return river.JobSnooze(delay + jitter)
	}
	return err
}

// InsertMany stores only trace IDs/sampling context beside existing metadata,
// so downstream work can join its originating HTTP request or cron trace.
func (m *jobLogMiddleware) InsertMany(ctx context.Context, params []*rivertype.JobInsertParams, inner func(context.Context) ([]*rivertype.JobInsertResult, error)) ([]*rivertype.JobInsertResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "queue.publish", "River job insert")
	carrier := telemetry.Propagation(ctx)
	if carrier.Trace != "" {
		encoded, _ := json.Marshal(carrier)
		for _, param := range params {
			metadata := map[string]json.RawMessage{}
			if len(param.Metadata) > 0 && json.Unmarshal(param.Metadata, &metadata) != nil {
				continue
			}
			if metadata == nil {
				metadata = map[string]json.RawMessage{}
			}
			metadata["_sentry"] = encoded
			if value, err := json.Marshal(metadata); err == nil {
				param.Metadata = value
			}
		}
	}
	started := time.Now()
	results, err := inner(ctx)
	telemetry.Observe(ctx, "queue.insert", time.Since(started), err)
	for _, result := range results {
		if result == nil || result.Job == nil {
			continue
		}
		outcome := "inserted"
		if result.UniqueSkippedAsDuplicate {
			outcome = "duplicate"
		}
		telemetry.Count(ctx, "queue.enqueued", 1, telemetry.Attributes("queue", result.Job.Queue, "outcome", outcome))
	}
	telemetry.FinishSpan(span, err)
	return results, err
}

var (
	_ rivertype.WorkerMiddleware    = (*jobLogMiddleware)(nil)
	_ rivertype.JobInsertMiddleware = (*jobLogMiddleware)(nil)
)
