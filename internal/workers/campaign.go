package workers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/campaign"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/riverqueue/river"
)

// CampaignProcessingWorker recomputes one campaign.
type CampaignProcessingWorker struct {
	river.WorkerDefaults[queue.CampaignProcessingArgs]
	Deps *Deps
}

func (w *CampaignProcessingWorker) Work(ctx context.Context, job *river.Job[queue.CampaignProcessingArgs]) error {
	id := job.Args.CampaignID
	var paused bool
	err := w.Deps.Pool.QueryRow(ctx, `
        SELECT coalesce(processing_paused, false)
        FROM campaigns WHERE campaign_id = $1`, id).Scan(&paused)
	if errors.Is(err, pgx.ErrNoRows) || paused {
		// A stale queued job must not alter the processing diagnostics of a
		// deleted or deliberately paused campaign.
		return nil
	}
	if err != nil {
		return err
	}

	started := time.Now().UTC()
	if _, err := w.Deps.Pool.Exec(ctx, `
        UPDATE campaigns
        SET last_processing_started_at = $2,
            last_processing_error = NULL
        WHERE campaign_id = $1`,
		id,
		started,
	); err != nil {
		return err
	}

	res, err := campaign.Process(ctx, w.Deps.Pool, id)
	duration := time.Since(started).Round(time.Millisecond)
	if errors.Is(err, campaign.ErrPaused) {
		// Somebody took this campaign out of processing deliberately. Retrying
		// would fight that decision every hour.
		return river.JobCancel(err)
	}
	if err != nil {
		message := err.Error()
		if len(message) > 1000 {
			message = message[:1000]
		}
		timedOut := campaignPostgresErrorCode(err, "57014") ||
			strings.Contains(strings.ToLower(message), "statement timeout")
		note := any(nil)
		if timedOut {
			note = "Processing exceeded the 10-minute safety limit. Narrow the campaign before resuming."
		}
		if _, updateErr := w.Deps.Pool.Exec(ctx, `
            UPDATE campaigns
            SET last_processing_duration_ms = $2,
                last_processing_error = $3,
                processing_paused = CASE WHEN $4 THEN true ELSE processing_paused END,
                processing_note = CASE WHEN $4 THEN $5 ELSE processing_note END
            WHERE campaign_id = $1`,
			id,
			duration.Milliseconds(),
			message,
			timedOut,
			note,
		); updateErr != nil {
			return errors.Join(err, updateErr)
		}
		if timedOut {
			// The campaign is now visibly paused; retrying the same query would
			// only consume another ten minutes.
			return nil
		}
		return err
	}
	if res == nil {
		// Deleted between dispatch and now.
		return nil
	}

	_, err = w.Deps.Pool.Exec(ctx, `
        UPDATE campaigns
        SET last_processing_duration_ms = $2,
            last_processing_killmails = $3,
            last_processing_error = NULL
        WHERE campaign_id = $1`,
		id,
		duration.Milliseconds(),
		res.Killmails,
	)
	return err
}

func campaignPostgresErrorCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

// cronCampaignPending computes campaigns that have never been computed.
//
// Separate from the hourly sweep and running every minute, because a campaign
// someone just created shows nothing until its first compute — an hour of an
// empty page is the difference between the feature working and appearing broken.
func (d *Deps) cronCampaignPending(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("campaign_pending")
	}

	rows, err := d.Pool.Query(ctx, `
        SELECT campaign_id FROM campaigns
        WHERE status = $1 AND processing_paused IS NOT TRUE
        ORDER BY created_at`, campaign.StatusPending)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var args []river.JobArgs
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		args = append(args, queue.CampaignProcessingArgs{CampaignID: id})
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(args) == 0 {
		return "", nil
	}

	// Immediate: somebody is watching a page that is currently empty.
	n, err := queue.DispatchMany(ctx, d.Queue, args, queue.Immediate)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d new campaigns queued", n), nil
}

// cronCampaignProcessing sweeps the active campaigns.
//
// Three jobs in one pass: recompute the ones with new activity, finalise the
// ones that have ended, and retire the ones nobody is fighting in. The gate
// check is what makes the sweep cheap — an idle campaign costs a few index
// probes rather than a recompute.
func (d *Deps) cronCampaignProcessing(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("campaign_processing")
	}

	rows, err := d.Pool.Query(ctx, `
        SELECT campaign_id FROM campaigns
        WHERE status = $1 AND processing_paused IS NOT TRUE`, campaign.StatusActive)
	if err != nil {
		return "", err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return "", err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", err
	}

	now := time.Now().UTC()
	var queued, finished, archived int
	var args []river.JobArgs

	for _, id := range ids {
		c, entities, err := campaign.Load(ctx, d.Pool, id)
		if err != nil {
			return "", err
		}

		// A campaign past its end plus grace is finished with — but only once
		// its statistics actually cover the window it claimed. Archiving before
		// that would freeze a campaign on numbers missing its final minutes,
		// and for a campaign with a prize pool those numbers decide who gets
		// paid. So the grace expiring queues the last compute; the sweep after
		// it, seeing the stats now cover the end, settles and archives.
		if c.Finished(now) {
			covered := c.ProcessedThrough != nil && c.EndTime != nil &&
				!c.ProcessedThrough.Before(*c.EndTime)
			if !covered {
				args = append(args, queue.CampaignProcessingArgs{CampaignID: id})
				queued++
				continue
			}

			if c.HasPrizePool {
				// Real ISK divided by the standings that were just confirmed
				// complete. Once this returns the pool is no longer live.
				if _, err := campaign.Finalize(ctx, d.Pool, id); err != nil {
					return "", fmt.Errorf("settle prize pool for campaign %s: %w", id, err)
				}
			}

			if _, err := d.Pool.Exec(ctx,
				`UPDATE campaigns SET status = $2, updated_at = now() WHERE campaign_id = $1`,
				id, campaign.StatusArchived); err != nil {
				return "", err
			}
			finished++
			continue
		}

		// An open-ended campaign nobody has fought in for a month is retired.
		// Area campaigns are exempt: a quiet region is still a region, and its
		// campaign is a standing watch rather than a dead one.
		if c.EndTime == nil && !campaign.IsAreaCampaign(entities, c.Location) {
			last := c.LastActivityAt
			if last == nil {
				last = &c.CreatedAt
			}
			if now.Sub(*last) > campaign.InactivityArchiveDays*24*time.Hour {
				if _, err := d.Pool.Exec(ctx,
					`UPDATE campaigns SET status = $2, updated_at = now() WHERE campaign_id = $1`,
					id, campaign.StatusArchived); err != nil {
					return "", err
				}
				archived++
				continue
			}
		}

		// The gate: only recompute what has actually changed.
		newKills, err := campaign.HasNewKills(ctx, d.Pool, c, entities)
		if err != nil {
			return "", err
		}
		if newKills {
			args = append(args, queue.CampaignProcessingArgs{CampaignID: id})
			queued++
		}
	}

	if _, err := queue.DispatchMany(ctx, d.Queue, args, queue.RecentBackfill); err != nil {
		return "", err
	}
	if queued == 0 && finished == 0 && archived == 0 {
		return "", nil
	}
	return fmt.Sprintf("%d of %d recomputed, %d finished, %d archived",
		queued, len(ids), finished, archived), nil
}
