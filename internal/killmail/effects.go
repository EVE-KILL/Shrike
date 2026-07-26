package killmail

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The derived-effect ledger.
//
// Storing a killmail is one write; everything the site shows about it is a
// dozen more — counters, rollups, the graph, the live feed. Those are not part
// of the insert and must not be, because a failure in any of them would roll
// back a killmail that was fetched, parsed and valued correctly.
//
// So each is a separate effect with a bit in killmail_processing.effects_completed,
// and each is retried on its own. The bit is what makes retrying safe: an
// effect that adds to a counter would double-count if it ran twice, and the
// ledger is the only thing standing between a retry and a wrong number.
//
// Both processes read this column. The TypeScript backend decides what to
// replay from it, so a killmail this code inserts and leaves at zero is one
// the TypeScript will happily re-aggregate on top of the work already done.
// That is why the bits are set here rather than left for later.

// Effect is one bit in the ledger.
type Effect int32

// Effect bits.
//
// These are a stored contract, not an internal enum: production rows hold
// these values indefinitely and the TypeScript backend reads the same column
// with the same meanings. A bit may be retired but must never be reused for a
// different effect — doing so would silently mark old killmails as having
// completed work they have never seen.
const (
	EffectLastActive             Effect = 1 << 0
	EffectWarInteractions        Effect = 1 << 1
	EffectDailyKillRollup        Effect = 1 << 2
	EffectAchievementsDispatched Effect = 1 << 3
	EffectFitDispatched          Effect = 1 << 4
	EffectStatsDispatched        Effect = 1 << 5
	EffectGraphDispatched        Effect = 1 << 6
	EffectEntitiesEnsured        Effect = 1 << 7
	EffectEventPublished         Effect = 1 << 8
	EffectTickerEvaluated        Effect = 1 << 9
	EffectStatsWritten           Effect = 1 << 10
)

// AllEffects is every bit above.
const AllEffects = EffectLastActive | EffectWarInteractions | EffectDailyKillRollup |
	EffectAchievementsDispatched | EffectFitDispatched | EffectStatsDispatched |
	EffectGraphDispatched | EffectEntitiesEnsured | EffectEventPublished |
	EffectTickerEvaluated | EffectStatsWritten

// AllEffectsExceptWar marks a killmail that has been through the pipeline but
// has only just been associated with a war.
//
// The war queue finds historical kills and attaches a war_id to them years
// after the fact. Everything else about such a killmail was aggregated long
// ago; only the war interaction is genuinely outstanding. Marking the rest
// complete is what stops the repair path re-running a full pipeline over a
// killmail whose only new fact is which war it belongs to.
const AllEffectsExceptWar = AllEffects &^ EffectWarInteractions

// IsComplete reports whether an effect has already run.
func IsComplete(mask, effect Effect) bool {
	return mask&effect == effect
}

// EffectOptions tunes how one effect runs.
type EffectOptions struct {
	// AllowUntracked runs the work even when no ledger row exists.
	//
	// Without a row there is nothing to record completion against, so the work
	// cannot be replay-protected — only for effects that are naturally
	// idempotent.
	AllowUntracked bool

	// BeforeLedgerLock runs inside the transaction before the ledger row is
	// locked. The war aggregation uses it to take a shared advisory lock, so
	// that a concurrent atomic rebuild cannot swap the table out from under an
	// in-flight increment.
	BeforeLedgerLock func(ctx context.Context, tx pgx.Tx) error
}

// RunDBEffect runs an effect that writes to Postgres, recording completion in
// the same transaction.
//
// Committing the work and the bit together is what makes the guarantee hold:
// there is no window in which the counter has been incremented but the ledger
// does not say so, and therefore no way for a retry to increment it twice.
//
// Returns true only when the work actually ran. A work function returning
// false means "not yet" — the effect stays pending and is retried later,
// which is how a killmail belonging to a war we have not imported waits for
// the war rather than being marked done without it.
func RunDBEffect(
	ctx context.Context,
	pool *pgxpool.Pool,
	killmailID int64,
	effect Effect,
	work func(ctx context.Context, tx pgx.Tx) (bool, error),
	opts EffectOptions,
) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if opts.BeforeLedgerLock != nil {
		if err := opts.BeforeLedgerLock(ctx, tx); err != nil {
			return false, err
		}
	}

	// FOR UPDATE rather than a plain read: two workers holding the same
	// killmail would otherwise both see the bit clear and both do the work.
	var completed Effect
	err = tx.QueryRow(ctx, `
        SELECT effects_completed FROM killmail_processing
        WHERE killmail_id = $1 FOR UPDATE`, killmailID).Scan(&completed)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if !opts.AllowUntracked {
			return false, nil
		}
		ran, err := work(ctx, tx)
		if err != nil || !ran {
			return false, err
		}
		return true, tx.Commit(ctx)
	case err != nil:
		return false, err
	}

	if IsComplete(completed, effect) {
		return false, nil
	}

	ran, err := work(ctx, tx)
	if err != nil || !ran {
		return false, err
	}

	if _, err := tx.Exec(ctx, `
        UPDATE killmail_processing
        SET effects_completed = effects_completed | $2, updated_at = now()
        WHERE killmail_id = $1`, killmailID, int32(effect)); err != nil {
		return false, fmt.Errorf("record effect %d for killmail %d: %w", effect, killmailID, err)
	}
	return true, tx.Commit(ctx)
}

// RunExternalEffect runs an effect that cannot share a Postgres transaction —
// enqueueing a job, publishing to the relay, calling ESI.
//
// The bit is set after the work succeeds, so a crash in between re-runs it.
// That is acceptable only because every such effect is addressed by a stable
// downstream id: the same killmail always produces the same job key, and
// River's uniqueness collapses the duplicate. An effect without that property
// does not belong here.
func RunExternalEffect(
	ctx context.Context,
	pool *pgxpool.Pool,
	killmailID int64,
	effect Effect,
	work func(ctx context.Context) error,
) (bool, error) {
	var completed Effect
	err := pool.QueryRow(ctx,
		`SELECT effects_completed FROM killmail_processing WHERE killmail_id = $1`,
		killmailID).Scan(&completed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if IsComplete(completed, effect) {
		return false, nil
	}

	if err := work(ctx); err != nil {
		return false, err
	}

	if _, err := pool.Exec(ctx, `
        UPDATE killmail_processing
        SET effects_completed = effects_completed | $2, updated_at = now()
        WHERE killmail_id = $1`, killmailID, int32(effect)); err != nil {
		return false, fmt.Errorf("record effect %d for killmail %d: %w", effect, killmailID, err)
	}
	return true, nil
}

// Tracked is what the pipeline needs to know about a killmail already stored.
type Tracked struct {
	Exists    bool
	Tracked   bool
	Completed Effect
	WarID     int32
}

// Pending is the set of effects still outstanding.
func (t Tracked) Pending() Effect { return AllEffects &^ t.Completed }

// Done reports a killmail with nothing left to do.
func (t Tracked) Done() bool { return t.Completed&AllEffects == AllEffects }

// ErrWarConflict means a killmail is already attached to a different war.
var ErrWarConflict = errors.New("killmail already assigned to a different war")

// Prepare inspects a stored killmail and, when a war id is offered, attaches it.
//
// The war attachment is the interesting half. A killmail that has been in the
// database for years can be discovered to belong to a war, and when that
// happens exactly one effect becomes outstanding again. Clearing that single
// bit — rather than reprocessing — is what keeps the war repair path cheap
// enough to run over a backlog.
func Prepare(ctx context.Context, pool *pgxpool.Pool, killmailID int64, requestedWarID int32) (Tracked, error) {
	var out Tracked

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var warID, processingID *int64
	var completed *int32
	err = tx.QueryRow(ctx, `
        SELECT k.war_id, p.killmail_id, p.effects_completed
        FROM killmails k
        LEFT JOIN killmail_processing p ON p.killmail_id = k.killmail_id
        WHERE k.killmail_id = $1
        FOR UPDATE OF k`, killmailID).Scan(&warID, &processingID, &completed)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}

	out.Exists = true
	out.Tracked = processingID != nil
	if out.Tracked && completed != nil {
		out.Completed = Effect(*completed)
	}
	if warID != nil {
		out.WarID = int32(*warID)
	}

	if requestedWarID != 0 {
		if out.WarID != 0 && out.WarID != requestedWarID {
			return out, fmt.Errorf("%w: killmail %d is in war %d, not %d",
				ErrWarConflict, killmailID, out.WarID, requestedWarID)
		}
		if out.WarID == 0 {
			if _, err := tx.Exec(ctx,
				`UPDATE killmails SET war_id = $2 WHERE killmail_id = $1`,
				killmailID, requestedWarID); err != nil {
				return out, err
			}
			out.WarID = requestedWarID

			if out.Tracked {
				if _, err := tx.Exec(ctx, `
                    UPDATE killmail_processing
                    SET effects_completed = effects_completed & ~$2::integer, updated_at = now()
                    WHERE killmail_id = $1`, killmailID, int32(EffectWarInteractions)); err != nil {
					return out, err
				}
				out.Completed &^= EffectWarInteractions
			}
		}

		// Archive importers deliberately do not create ledger rows. A war
		// archive can nevertheless have assigned this exact war_id already, so
		// the adoption cannot live only inside the "war was NULL" branch above.
		// The war queue's replay predicate selected this row precisely because
		// its interaction is missing; mark every unrelated historical effect
		// complete and leave only that one pending.
		if !out.Tracked {
			if _, err := tx.Exec(ctx, `
                INSERT INTO killmail_processing (killmail_id, effects_completed)
                VALUES ($1, $2)`, killmailID, int32(AllEffectsExceptWar)); err != nil {
				return out, err
			}
			out.Tracked = true
			out.Completed = AllEffectsExceptWar
		}
	}

	return out, tx.Commit(ctx)
}
