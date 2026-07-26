package workers

import (
	"context"
	"time"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/wars"
	"github.com/riverqueue/river"
)

// WarWorker fetches one war's metadata, its allies, and the killmails fought
// under it.
type WarWorker struct {
	river.WorkerDefaults[queue.WarArgs]
	Deps *Deps
}

func (w *WarWorker) Work(ctx context.Context, job *river.Job[queue.WarArgs]) error {
	id := job.Args.WarID

	// Read what we believed before the refresh overwrites it — a war that
	// started or ended is only news relative to the last thing we knew.
	prior, err := w.priorState(ctx, id)
	if err != nil {
		return err
	}

	war, err := wars.Refresh(ctx, w.Deps.Pool, w.Deps.ESI, id)
	if err != nil {
		return err
	}
	// ESI does not recognise the id — nothing stored, nothing to follow up.
	if war == nil {
		return nil
	}

	w.announceWar(ctx, id, prior, *war, time.Now().UTC())

	if _, err := wars.StoreAllies(ctx, w.Deps.Pool, id, *war); err != nil {
		return err
	}

	// A metadata-only refresh stops here. The repair sweep that fills in wars
	// referenced by stored killmails already has those killmails, so paging
	// through ESI's list for them would spend the request and error budget to
	// discover nothing — and there are hundreds of thousands of finished wars
	// to repair.
	if job.Args.MetadataOnly || w.Deps.Queue == nil {
		return nil
	}

	missing, err := wars.MissingKillmails(ctx, w.Deps.Pool, w.Deps.ESI, id)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}

	batch := make([]river.JobArgs, 0, len(missing))
	for _, ref := range missing {
		// The war id travels with the job: the public killmail endpoint does
		// not return it, and this is the only place that knows it.
		batch = append(batch, queue.KillmailArgs{
			KillmailID:   ref.KillmailID,
			KillmailHash: ref.KillmailHash,
			WarID:        id,
		})
	}

	_, err = queue.DispatchMany(ctx, w.Deps.Queue, batch, queue.DormantBackfill)
	return err
}
