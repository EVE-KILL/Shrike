package queue

import (
	"context"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// Dispatching work.
//
// Every helper here goes through InsertOptsFor, so a job's queue, retry budget
// and deduplication are decided in one place rather than at each call site.
// That is the property worth keeping: the TypeScript dispatcher accreted
// per-call `dedupe:` strings, and the ones that were forgotten are exactly the
// jobs that ran twice.

// InsertResult reports what an insert did.
type InsertResult struct {
	Job *rivertype.JobRow

	// Duplicate is true when an identical job was already queued and this
	// insert collapsed into it. Not an error — it is the deduplication working
	// — but callers counting dispatches need to tell the two apart.
	Duplicate bool
}

// Dispatch enqueues one job.
func Dispatch(ctx context.Context, c *Client, args river.JobArgs, priority Priority) (InsertResult, error) {
	res, err := c.Insert(ctx, args, InsertOptsFor(args.Kind(), priority))
	if err != nil {
		return InsertResult{}, err
	}
	if err := c.promoteDuplicate(ctx, res, priority); err != nil {
		return InsertResult{}, err
	}
	return InsertResult{Job: res.Job, Duplicate: res.UniqueSkippedAsDuplicate}, nil
}

// DispatchAt enqueues one job to run no earlier than the given delay.
//
// Used where the TypeScript passed a `delay`: history fetches are spread over a
// second or so after the character that triggered them, so a burst of killmails
// does not put a thousand history requests into ESI at the same instant.
func DispatchAt(ctx context.Context, c *Client, args river.JobArgs, priority Priority, after time.Duration) (InsertResult, error) {
	opts := InsertOptsFor(args.Kind(), priority)
	if after > 0 {
		opts.ScheduledAt = time.Now().Add(after)
	}
	res, err := c.Insert(ctx, args, opts)
	if err != nil {
		return InsertResult{}, err
	}
	if err := c.promoteDuplicate(ctx, res, priority); err != nil {
		return InsertResult{}, err
	}
	return InsertResult{Job: res.Job, Duplicate: res.UniqueSkippedAsDuplicate}, nil
}

// DispatchMany enqueues a batch in one round trip.
//
// Worth using wherever the count is unbounded — the killmail repair cron finds
// thousands of missing kills at a time, and inserting those one at a time is
// thousands of round trips for work that fits in a single statement.
func DispatchMany(ctx context.Context, c *Client, batch []river.JobArgs, priority Priority) (int, error) {
	if len(batch) == 0 {
		return 0, nil
	}

	params := make([]river.InsertManyParams, 0, len(batch))
	for _, args := range batch {
		params = append(params, river.InsertManyParams{
			Args:       args,
			InsertOpts: InsertOptsFor(args.Kind(), priority),
		})
	}

	results, err := c.InsertMany(ctx, params)
	if err != nil {
		return 0, err
	}

	inserted := 0
	for _, r := range results {
		if err := c.promoteDuplicate(ctx, r, priority); err != nil {
			return inserted, err
		}
		if !r.UniqueSkippedAsDuplicate {
			inserted++
		}
	}
	return inserted, nil
}

// promoteDuplicate moves an already-available duplicate into a more urgent
// lane when the new dispatch asks for one.
//
// This is the River equivalent of QueueManager.promoteStoredJob. Without it, a
// million-row dormant backfill can enqueue an entity first and a live killmail
// discovering the same entity remains stuck behind the backfill. Only
// available jobs are changed: running work cannot be reprioritised, and a
// deliberately scheduled job must keep its delay.
func (c *Client) promoteDuplicate(
	ctx context.Context,
	res *rivertype.JobInsertResult,
	requested Priority,
) error {
	if c == nil || c.pool == nil || res == nil || res.Job == nil ||
		!res.UniqueSkippedAsDuplicate || !requested.Valid() ||
		res.Job.Priority <= int(requested) {
		return nil
	}

	tag, err := c.pool.Exec(ctx, `
        UPDATE river_job
        SET priority = $2
        WHERE id = $1
          AND state = $3
          AND priority > $2`,
		res.Job.ID, int(requested), rivertype.JobStateAvailable)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		res.Job.Priority = int(requested)
	}
	return nil
}

// DispatchCascade enqueues the follow-up work an entity refresh discovered.
//
// The cascade is already deduplicated and filtered by the entities package, so
// everything here is worth fetching. It inherits the parent's tier rather than
// resetting to the top, or a single dormant backfill job would spawn immediate
// work and the priority tiers would stop meaning anything.
func DispatchCascade(ctx context.Context, c *Client, cascade Cascade, parent Priority) (int, error) {
	tier := CascadePriority(parent)

	var batch []river.JobArgs
	for _, id := range cascade.Characters {
		batch = append(batch, CharacterArgs{CharacterID: id})
	}
	for _, id := range cascade.Corporations {
		batch = append(batch, CorporationArgs{CorporationID: id})
	}
	for _, id := range cascade.Alliances {
		batch = append(batch, AllianceArgs{AllianceID: id})
	}
	for _, id := range cascade.CharacterHistories {
		batch = append(batch, CharacterHistoryArgs{CharacterID: id})
	}
	for _, id := range cascade.CorporationHistories {
		batch = append(batch, CorporationHistoryArgs{CorporationID: id})
	}

	// River rejects a mixed-queue batch, so each kind goes separately. Grouping
	// by kind rather than inserting one at a time keeps this to five round
	// trips regardless of how many entities a large fight named.
	total := 0
	for _, group := range groupByKind(batch) {
		n, err := DispatchMany(ctx, c, group, tier)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

// Cascade is the follow-up work an entity refresh implies.
//
// Structurally identical to entities.Cascade, and deliberately not that type:
// having the queue import the entities package and the entities package import
// the queue would be a cycle, and the alternative — entities knowing how to
// enqueue — is exactly the coupling that package's doc comment refuses.
type Cascade struct {
	Characters           []int32
	Corporations         []int32
	Alliances            []int32
	CharacterHistories   []int32
	CorporationHistories []int32
}

func groupByKind(batch []river.JobArgs) [][]river.JobArgs {
	byKind := map[string][]river.JobArgs{}
	var order []string
	for _, a := range batch {
		k := a.Kind()
		if _, seen := byKind[k]; !seen {
			order = append(order, k)
		}
		byKind[k] = append(byKind[k], a)
	}

	out := make([][]river.JobArgs, 0, len(order))
	for _, k := range order {
		out = append(out, byKind[k])
	}
	return out
}
