package workers

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
)

// The entity workers are thin, and that is the design: the fetching, storing
// and cascade logic all live in internal/entities, tested there against a real
// database. What is left here is translating a job into a call and a result
// back into more jobs.
//
// In particular there is no error classification here, because there is nothing
// left to classify. The Refresher already distinguishes the two cases that
// matter: ESI answering "this does not exist" is recorded as deleted and
// returns no error, so the job succeeds and the id is never retried, while an
// unreachable or unwell ESI comes back as entities.ErrTransient. Everything
// that reaches these workers as an error is therefore worth retrying, and
// River's default handling is correct.

// CharacterWorker fetches one character.
type CharacterWorker struct {
	river.WorkerDefaults[queue.CharacterArgs]
	Deps *Deps
}

func (w *CharacterWorker) Work(ctx context.Context, job *river.Job[queue.CharacterArgs]) error {
	res, err := w.Deps.refresher().Character(ctx, job.Args.CharacterID)
	if err != nil {
		return err
	}

	// History is spread over the following second or so rather than dispatched
	// with the parent. A large fight resolves hundreds of characters at once,
	// and without the spread every one of their history fetches hits ESI in the
	// same instant — a burst that the rate limiter then has to absorb by
	// stalling everything else behind it.
	if len(res.Cascade.CharacterHistories) > 0 && w.Deps.Queue != nil {
		delay := time.Duration(100+rand.IntN(1400)) * time.Millisecond //nolint:gosec // jitter
		for _, id := range res.Cascade.CharacterHistories {
			if _, err := queue.DispatchAt(ctx, w.Deps.Queue,
				queue.CharacterHistoryArgs{CharacterID: id},
				cascadeTier(job.Priority), delay); err != nil {
				return err
			}
		}
		res.Cascade.CharacterHistories = nil
	}

	_, err = w.Deps.dispatchCascade(ctx, res.Cascade, cascadeTier(job.Priority))
	return err
}

// CorporationWorker fetches one corporation.
type CorporationWorker struct {
	river.WorkerDefaults[queue.CorporationArgs]
	Deps *Deps
}

func (w *CorporationWorker) Work(ctx context.Context, job *river.Job[queue.CorporationArgs]) error {
	res, err := w.Deps.refresher().Corporation(ctx, job.Args.CorporationID)
	if err != nil {
		return err
	}
	_, err = w.Deps.dispatchCascade(ctx, res.Cascade, cascadeTier(job.Priority))
	return err
}

// AllianceWorker fetches one alliance.
type AllianceWorker struct {
	river.WorkerDefaults[queue.AllianceArgs]
	Deps *Deps
}

func (w *AllianceWorker) Work(ctx context.Context, job *river.Job[queue.AllianceArgs]) error {
	res, err := w.Deps.refresher().Alliance(ctx, job.Args.AllianceID)
	if err != nil {
		return err
	}
	_, err = w.Deps.dispatchCascade(ctx, res.Cascade, cascadeTier(job.Priority))
	return err
}

// CharacterHistoryWorker fetches one character's corporation history.
type CharacterHistoryWorker struct {
	river.WorkerDefaults[queue.CharacterHistoryArgs]
	Deps *Deps
}

func (w *CharacterHistoryWorker) Work(ctx context.Context, job *river.Job[queue.CharacterHistoryArgs]) error {
	_, err := w.Deps.refresher().CharacterHistory(ctx, job.Args.CharacterID, job.Args.Force)
	return err
}

// CorporationHistoryWorker fetches one corporation's alliance history.
type CorporationHistoryWorker struct {
	river.WorkerDefaults[queue.CorporationHistoryArgs]
	Deps *Deps
}

func (w *CorporationHistoryWorker) Work(ctx context.Context, job *river.Job[queue.CorporationHistoryArgs]) error {
	_, err := w.Deps.refresher().CorporationHistory(ctx, job.Args.CorporationID, job.Args.Force)
	return err
}

// cascadeTier maps a River priority back onto a tier.
//
// River hands the worker the numeric priority it was inserted with, and the
// follow-up work has to inherit it — otherwise a dormant backfill's children
// enter at the top tier and overtake the live feed. An unrecognised value falls
// back to Live rather than Immediate.
func cascadeTier(priority int) queue.Priority {
	p := queue.Priority(priority)
	if !p.Valid() {
		return queue.Live
	}
	return queue.CascadePriority(p)
}
