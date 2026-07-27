package workers

import (
	"context"
	"fmt"

	"github.com/eve-kill/shrike/internal/images"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
)

// ImageRefreshWorker moves CCP refreshes off the HTTP request path. River's
// active-state uniqueness means a traffic spike can never multiply the origin
// request or write the same B2 pointer concurrently.
type ImageRefreshWorker struct {
	river.WorkerDefaults[queue.ImageRefreshArgs]
	Deps *Deps
}

func (w *ImageRefreshWorker) Work(
	ctx context.Context,
	job *river.Job[queue.ImageRefreshArgs],
) error {
	if w.Deps.Images == nil || !w.Deps.Images.Available() {
		return fmt.Errorf("image storage is not configured")
	}
	return w.Deps.Images.RefreshEntity(
		ctx,
		images.EntityKind(job.Args.EntityKind),
		job.Args.EntityID,
	)
}

// cronImageTypeSync checks GitHub once per day and mirrors a changed
// TurtleTools Image Export Collection into direct type-ID keys in B2. The
// hash manifest is synchronization bookkeeping only and is not read while
// serving.
// An unconfigured image bucket is a supported deployment mode, not a failed
// cron that should burn retries forever.
func (d *Deps) cronImageTypeSync(ctx context.Context) (string, error) {
	if d.ImageStore == nil {
		return "image storage is not configured; skipped", nil
	}
	result, err := images.SyncTypeExport(
		ctx,
		d.ImageStore,
		images.TypeExportSyncOptions{
			Token: d.GitHubToken, UserAgent: d.UserAgent,
		},
	)
	if err != nil {
		return "", err
	}
	if !result.Changed {
		return fmt.Sprintf("TurtleTools %s is already current", result.Release), nil
	}
	return fmt.Sprintf(
		"published TurtleTools %s: %d objects, %d bytes",
		result.Release,
		result.Import.Uploaded,
		result.Import.Bytes,
	), nil
}
