package maintenance

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

const (
	DefaultEntityBatch     = 500
	DefaultEntityStaleDays = 30
)

var eveLaunch = time.Date(2003, 1, 1, 0, 0, 0, 0, time.UTC)

type StaleEntityOptions struct {
	SkipAlliances    bool
	SkipCorporations bool
	SkipCharacters   bool
	AllianceDays     int
	CorporationDays  int
	Batch            int
	Limit            int
	DryRun           bool
}

type EntityQueueResult struct {
	AllianceCandidates    int64 `json:"alliance_candidates"`
	AllianceQueued        int64 `json:"alliance_queued"`
	CorporationCandidates int64 `json:"corporation_candidates"`
	CorporationQueued     int64 `json:"corporation_queued"`
	CharacterCandidates   int64 `json:"character_candidates"`
	CharacterQueued       int64 `json:"character_queued"`
}

func (o *StaleEntityOptions) normalize() {
	if o.AllianceDays < 1 {
		o.AllianceDays = DefaultEntityStaleDays
	}
	if o.CorporationDays < 1 {
		o.CorporationDays = DefaultEntityStaleDays
	}
	if o.Batch < 1 {
		o.Batch = DefaultEntityBatch
	}
}

// QueueStaleEntities reproduces queue:stale-entities' three intentionally
// different repair sets:
//
//   - alliances and corporations are selected by updated_at age;
//   - NPC corporations are excluded;
//   - characters are selected only when birthday is missing or predates EVE.
//
// Ordinary stale characters are refreshed by the live affiliation/entity
// cadence. Treating every stale character as a metadata repair turns this
// maintenance command into millions of unnecessary ESI requests.
func QueueStaleEntities(
	ctx context.Context,
	pool *pgxpool.Pool,
	client *queue.Client,
	opts StaleEntityOptions,
) (EntityQueueResult, error) {
	var out EntityQueueResult
	opts.normalize()

	if !opts.SkipCharacters && !opts.DryRun && opts.Limit <= 0 {
		return out, fmt.Errorf("refusing unbounded character dispatch: pass --limit or --skip-characters")
	}
	if !opts.DryRun && client == nil {
		return out, fmt.Errorf("queue client is required outside dry-run mode")
	}

	now := time.Now().UTC()
	if !opts.SkipAlliances {
		n, queued, err := queueEntitySet(ctx, pool, client, entitySet{
			table: "alliances", column: "alliance_id",
			where: "deleted IS NOT TRUE AND (updated_at IS NULL OR updated_at < $1)",
			arg:   now.AddDate(0, 0, -opts.AllianceDays),
			build: func(id int32) river.JobArgs { return queue.AllianceArgs{AllianceID: id} },
		}, opts)
		if err != nil {
			return out, fmt.Errorf("queue alliances: %w", err)
		}
		out.AllianceCandidates, out.AllianceQueued = n, queued
	}

	if !opts.SkipCorporations {
		n, queued, err := queueEntitySet(ctx, pool, client, entitySet{
			table: "corporations", column: "corporation_id",
			where: fmt.Sprintf(`deleted IS NOT TRUE
				AND corporation_id >= %d
				AND (updated_at IS NULL OR updated_at < $1)`, entities.PlayerCorporationIDMin),
			arg:   now.AddDate(0, 0, -opts.CorporationDays),
			build: func(id int32) river.JobArgs { return queue.CorporationArgs{CorporationID: id} },
		}, opts)
		if err != nil {
			return out, fmt.Errorf("queue corporations: %w", err)
		}
		out.CorporationCandidates, out.CorporationQueued = n, queued
	}

	if !opts.SkipCharacters {
		n, queued, err := queueEntitySet(ctx, pool, client, entitySet{
			table: "characters", column: "character_id",
			where: "deleted IS NOT TRUE AND (birthday IS NULL OR birthday < $1)",
			arg:   eveLaunch,
			build: func(id int32) river.JobArgs { return queue.CharacterArgs{CharacterID: id} },
		}, opts)
		if err != nil {
			return out, fmt.Errorf("queue characters: %w", err)
		}
		out.CharacterCandidates, out.CharacterQueued = n, queued
	}

	return out, nil
}

type entitySet struct {
	table  string
	column string
	where  string
	arg    any
	build  func(int32) river.JobArgs
}

func queueEntitySet(
	ctx context.Context,
	pool *pgxpool.Pool,
	client *queue.Client,
	set entitySet,
	opts StaleEntityOptions,
) (candidates, queued int64, err error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = math.MaxInt32
	}

	if err := pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT LEAST(count(*), $2)::bigint
		FROM %s
		WHERE %s`, set.table, set.where), set.arg, limit).Scan(&candidates); err != nil {
		return 0, 0, err
	}
	if opts.DryRun || candidates == 0 {
		return candidates, 0, nil
	}

	var cursor int32
	remaining := limit
	for remaining > 0 {
		batchSize := min(opts.Batch, remaining)
		rows, qerr := pool.Query(ctx, fmt.Sprintf(`
			SELECT %s
			FROM %s
			WHERE %s AND %s > $2
			ORDER BY %s
			LIMIT $3`, set.column, set.table, set.where, set.column, set.column),
			set.arg, cursor, batchSize)
		if qerr != nil {
			return candidates, queued, qerr
		}

		var jobs []river.JobArgs
		for rows.Next() {
			var id int32
			if scanErr := rows.Scan(&id); scanErr != nil {
				rows.Close()
				return candidates, queued, scanErr
			}
			cursor = id
			jobs = append(jobs, set.build(id))
		}
		rows.Close()
		if rows.Err() != nil {
			return candidates, queued, rows.Err()
		}
		if len(jobs) == 0 {
			break
		}

		inserted, dispatchErr := queue.DispatchMany(ctx, client, jobs, queue.DormantBackfill)
		if dispatchErr != nil {
			return candidates, queued, dispatchErr
		}
		queued += int64(inserted)
		remaining -= len(jobs)
		if len(jobs) < batchSize {
			break
		}
	}
	return candidates, queued, nil
}
