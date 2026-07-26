package wars

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Rebuilding war_interactions from the killmails.
//
// The incremental path adds one killmail at a time and is what keeps the table
// current. This is the authority it can be checked against: it derives every
// row from the killmails and attackers directly, so a table that has drifted —
// from a bug, an interrupted backfill, or the three months the aggregation was
// simply not running — can be replaced with what the data actually says.
//
// Two independent implementations of the same aggregation is a cost worth
// paying here. The incremental one is per killmail and cannot see the whole
// war; this one is set-based and cannot run per killmail. Agreeing is the
// property that matters, and the verification test asserts it.

// Summary describes what a scope currently holds.
type Summary struct {
	Killmails         int64
	KillmailISK       float64
	Rows              int64
	Wars              int64
	CombinedCorpKills int64
	CombinedCorpISK   float64
}

// Summarize reports the current state of a scope, for the preview.
//
// The combined corporation figures are the ones to read: they are one row per
// war per victim corporation, so they are directly comparable to the killmail
// count above them. If those two disagree, the table has drifted.
func Summarize(ctx context.Context, pool *pgxpool.Pool, warID int32) (Summary, error) {
	var s Summary

	killScope, interactionScope := "k.war_id IS NOT NULL", "TRUE"
	var args []any
	if warID != 0 {
		killScope, interactionScope = "k.war_id = $1", "i.war_id = $1"
		args = append(args, warID)
	}

	if err := pool.QueryRow(ctx, fmt.Sprintf(`
        SELECT count(*), coalesce(sum(k.total_value), 0)
        FROM killmails k
        JOIN wars w ON w.war_id = k.war_id
        WHERE %s`, killScope), args...).Scan(&s.Killmails, &s.KillmailISK); err != nil {
		return s, err
	}

	if err := pool.QueryRow(ctx, fmt.Sprintf(`
        SELECT count(*), count(DISTINCT i.war_id),
               coalesce(sum(i.count) FILTER (
                   WHERE i.side = 0 AND i.category = 0 AND i.target_type = 1), 0),
               coalesce(sum(i.isk_value) FILTER (
                   WHERE i.side = 0 AND i.category = 0 AND i.target_type = 1), 0)
        FROM war_interactions i
        WHERE %s`, interactionScope), args...).
		Scan(&s.Rows, &s.Wars, &s.CombinedCorpKills, &s.CombinedCorpISK); err != nil {
		return s, err
	}

	return s, nil
}

// RebuildResult reports what a rebuild produced.
type RebuildResult struct {
	Rows int64
	Wars int64
}

// Rebuild replaces war_interactions for a scope, atomically.
//
// Everything happens in one repeatable-read transaction holding the exclusive
// advisory lock, which is what makes it safe to run against a live killboard:
// incremental writers take the same lock shared, so none of them can slip an
// increment into a table that is about to be replaced.
//
// The replacement is built in a temporary table and validated before anything
// in production is touched. A unique index on the logical key is that
// validation — if the aggregation somehow produced two rows for one key, the
// index creation aborts the transaction while the real table is still intact.
//
// Pass warID 0 to rebuild every war.
func Rebuild(ctx context.Context, pool *pgxpool.Pool, warID int32) (RebuildResult, error) {
	var out RebuildResult

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	// A full rebuild scans every war kill there has ever been, so the statement
	// timeout has to go. The lock timeout stays: waiting a minute for the lock
	// is reasonable, waiting forever is a hung deploy.
	if _, err := tx.Exec(ctx, `SET LOCAL statement_timeout = 0`); err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '60s'`); err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, LockNamespace, LockKey); err != nil {
		return out, fmt.Errorf("take the war-interaction lock: %w", err)
	}

	scope := "k.war_id IS NOT NULL"
	var args []any
	if warID != 0 {
		scope = "k.war_id = $1"
		args = append(args, warID)
	}

	if _, err := tx.Exec(ctx, `
        CREATE TEMP TABLE war_interactions_rebuild (
            LIKE war_interactions INCLUDING DEFAULTS
        ) ON COMMIT DROP`); err != nil {
		return out, err
	}

	if _, err := tx.Exec(ctx, killedSQL(scope), args...); err != nil {
		return out, fmt.Errorf("aggregate killed direction: %w", err)
	}
	if _, err := tx.Exec(ctx, killedBySQL(scope), args...); err != nil {
		return out, fmt.Errorf("aggregate killed-by direction: %w", err)
	}

	// Validation and an insertion aid at once: a duplicate logical key aborts
	// here, before production rows have been touched.
	if _, err := tx.Exec(ctx, `
        CREATE UNIQUE INDEX war_interactions_rebuild_pk
        ON war_interactions_rebuild (war_id, side, category, target_type, target_id)`); err != nil {
		return out, fmt.Errorf("validate the replacement: %w", err)
	}

	if err := tx.QueryRow(ctx, `
        SELECT count(*), count(DISTINCT war_id) FROM war_interactions_rebuild`).
		Scan(&out.Rows, &out.Wars); err != nil {
		return out, err
	}

	if warID == 0 {
		if _, err := tx.Exec(ctx, `TRUNCATE war_interactions`); err != nil {
			return out, err
		}
	} else {
		if _, err := tx.Exec(ctx, `DELETE FROM war_interactions WHERE war_id = $1`, warID); err != nil {
			return out, err
		}
	}

	// Inserted in primary-key order so the index build behind it is sequential
	// rather than random.
	if _, err := tx.Exec(ctx, `
        INSERT INTO war_interactions (
            war_id, side, category, target_type, target_id,
            count, isk_value, last_killmail_id, last_killmail_time)
        SELECT war_id, side, category, target_type, target_id,
               count, isk_value, last_killmail_id, last_killmail_time
        FROM war_interactions_rebuild
        ORDER BY war_id, side, category, target_type, target_id`); err != nil {
		return out, err
	}

	// The rebuilt rows include every war killmail, so their war effect is now
	// genuinely complete. A killmail with no ledger row gets one marked fully
	// done — it predates the ledger and its other effects really did run — and
	// one already tracked keeps its pending non-war effects and gains only the
	// war bit.
	if _, err := tx.Exec(ctx, fmt.Sprintf(`
        INSERT INTO killmail_processing (killmail_id, effects_completed)
        SELECT k.killmail_id, %d
        FROM killmails k
        JOIN wars w ON w.war_id = k.war_id
        WHERE %s
        ON CONFLICT (killmail_id) DO UPDATE SET
            effects_completed = killmail_processing.effects_completed | %d,
            updated_at = now()`,
		allKillmailEffects, scope, effectWarInteractions), args...); err != nil {
		return out, fmt.Errorf("seed the effect ledger: %w", err)
	}

	// A killmail naming a war we have no row for could not be aggregated —
	// there are no sides to attribute it to. Clearing the bit leaves it pending
	// so the repair path replays it once the war is restored from ESI, rather
	// than marking it complete over a war that was never read.
	if _, err := tx.Exec(ctx, fmt.Sprintf(`
        UPDATE killmail_processing p
        SET effects_completed = p.effects_completed & ~%d,
            updated_at = now()
        FROM killmails k
        LEFT JOIN wars w ON w.war_id = k.war_id
        WHERE p.killmail_id = k.killmail_id
          AND k.war_id IS NOT NULL
          AND w.war_id IS NULL
          AND %s`, effectWarInteractions, scope), args...); err != nil {
		return out, fmt.Errorf("reopen unresolvable war effects: %w", err)
	}

	return out, tx.Commit(ctx)
}

// AggregateAllInTx runs the set-based aggregation straight into
// war_interactions, inside a transaction the caller controls.
//
// Exists for the agreement test, which has to compute both answers inside one
// transaction it will roll back — so it can use neither Rebuild's own
// transaction nor its exclusive lock. Sharing the SQL is the whole point: a
// test that reimplemented the aggregation would be testing itself.
func AggregateAllInTx(ctx context.Context, tx pgx.Tx) error {
	const scope = "k.war_id IS NOT NULL"
	if _, err := tx.Exec(ctx, strings.Replace(killedSQL(scope),
		"war_interactions_rebuild", "war_interactions", 1)); err != nil {
		return fmt.Errorf("aggregate killed direction: %w", err)
	}
	if _, err := tx.Exec(ctx, strings.Replace(killedBySQL(scope),
		"war_interactions_rebuild", "war_interactions", 1)); err != nil {
		return fmt.Errorf("aggregate killed-by direction: %w", err)
	}
	return nil
}

// The ledger bits, duplicated as constants rather than imported.
//
// internal/killmail imports nothing from here, and importing it from here would
// close a cycle. The values are a stored contract that cannot change anyway,
// and the effects test asserts them.
const (
	effectWarInteractions = 1 << 1
	allKillmailEffects    = 1<<11 - 1
)

// killedSQL aggregates the killed direction.
//
// One row per (war, side, victim entity): combined once per killmail, plus once
// for each declared side its attackers represent. The UNION against a constant
// zero side is what produces the combined row without a second pass.
func killedSQL(scope string) string {
	return fmt.Sprintf(`
        INSERT INTO war_interactions_rebuild (
            war_id, side, category, target_type, target_id,
            count, isk_value, last_killmail_id, last_killmail_time)
        WITH scoped_kills AS MATERIALIZED (
            SELECT k.* FROM killmails k
            JOIN wars scope_war ON scope_war.war_id = k.war_id
            WHERE %s
        ), attacker_sides AS (
            SELECT DISTINCT k.killmail_id, %s AS side
            FROM scoped_kills k
            JOIN killmail_attackers a ON a.killmail_id = k.killmail_id
            JOIN wars w ON w.war_id = k.war_id
        ), sides AS (
            SELECT killmail_id, 0::smallint AS side FROM scoped_kills
            UNION
            SELECT killmail_id, side FROM attacker_sides WHERE side IS NOT NULL
        ), contributions AS (
            SELECT k.war_id, s.side,
                   target.target_type, target.target_id,
                   coalesce(k.total_value, 0) AS isk_value,
                   k.killmail_id, k.killmail_time
            FROM scoped_kills k
            JOIN sides s USING (killmail_id)
            CROSS JOIN LATERAL (VALUES
                (0::smallint, k.victim_character_id),
                (1::smallint, k.victim_corporation_id),
                (2::smallint, k.victim_alliance_id)
            ) AS target(target_type, target_id)
            WHERE target.target_id IS NOT NULL
        ), ranked AS (
            SELECT *,
                   count(*) OVER (
                       PARTITION BY war_id, side, target_type, target_id
                   )::integer AS interaction_count,
                   sum(isk_value) OVER (
                       PARTITION BY war_id, side, target_type, target_id
                   ) AS total_isk_value,
                   row_number() OVER (
                       PARTITION BY war_id, side, target_type, target_id
                       ORDER BY killmail_time DESC, killmail_id DESC
                   ) AS latest_rank
            FROM contributions
        )
        SELECT war_id, side, 0::smallint, target_type, target_id,
               interaction_count, total_isk_value, killmail_id, killmail_time
        FROM ranked
        WHERE latest_rank = 1`,
		scope, sideExpr("a.alliance_id", "a.corporation_id"))
}

// killedBySQL aggregates the killed-by direction.
//
// The target is the final blow alone, and the side is the victim's: this row
// answers "what did this side lose ships to", so it hangs off the side that
// took the loss rather than the one that dealt it.
func killedBySQL(scope string) string {
	return fmt.Sprintf(`
        INSERT INTO war_interactions_rebuild (
            war_id, side, category, target_type, target_id,
            count, isk_value, last_killmail_id, last_killmail_time)
        WITH scoped_kills AS MATERIALIZED (
            SELECT k.* FROM killmails k
            JOIN wars scope_war ON scope_war.war_id = k.war_id
            WHERE %s
        ), final_blows AS (
            SELECT DISTINCT ON (a.killmail_id)
                   a.killmail_id, a.character_id, a.corporation_id, a.alliance_id
            FROM killmail_attackers a
            JOIN scoped_kills k ON k.killmail_id = a.killmail_id
            WHERE a.final_blow = true
            ORDER BY a.killmail_id, a.attacker_index
        ), victim_sides AS (
            SELECT k.killmail_id, 0::smallint AS side FROM scoped_kills k
            UNION
            SELECT k.killmail_id, %s AS side
            FROM scoped_kills k
            JOIN wars w ON w.war_id = k.war_id
        ), contributions AS (
            SELECT k.war_id, s.side,
                   target.target_type, target.target_id,
                   coalesce(k.total_value, 0) AS isk_value,
                   k.killmail_id, k.killmail_time
            FROM scoped_kills k
            JOIN final_blows fb USING (killmail_id)
            JOIN victim_sides s USING (killmail_id)
            CROSS JOIN LATERAL (VALUES
                (0::smallint, fb.character_id),
                (1::smallint, fb.corporation_id),
                (2::smallint, fb.alliance_id)
            ) AS target(target_type, target_id)
            WHERE s.side IS NOT NULL AND target.target_id IS NOT NULL
        ), ranked AS (
            SELECT *,
                   count(*) OVER (
                       PARTITION BY war_id, side, target_type, target_id
                   )::integer AS interaction_count,
                   sum(isk_value) OVER (
                       PARTITION BY war_id, side, target_type, target_id
                   ) AS total_isk_value,
                   row_number() OVER (
                       PARTITION BY war_id, side, target_type, target_id
                       ORDER BY killmail_time DESC, killmail_id DESC
                   ) AS latest_rank
            FROM contributions
        )
        SELECT war_id, side, 1::smallint, target_type, target_id,
               interaction_count, total_isk_value, killmail_id, killmail_time
        FROM ranked
        WHERE latest_rank = 1`,
		scope, sideExpr("k.victim_alliance_id", "k.victim_corporation_id"))
}

// sideExpr renders the side classification for a pair of columns.
//
// The same rule as Membership.Side, expressed in SQL: aggressor by direct
// match, defender by direct match or by being a war ally, NULL for anyone else.
// Written once and instantiated twice because the two directions classify
// different columns with identical logic, and having them drift is exactly the
// bug this file exists to detect.
func sideExpr(allianceCol, corporationCol string) string {
	return fmt.Sprintf(`CASE
        WHEN (%[1]s IS NOT NULL AND %[1]s = w.aggressor_alliance_id)
          OR (%[2]s IS NOT NULL AND %[2]s = w.aggressor_corporation_id)
            THEN 1::smallint
        WHEN (%[1]s IS NOT NULL AND (
                  %[1]s = w.defender_alliance_id
                  OR EXISTS (SELECT 1 FROM war_allies wa
                             WHERE wa.war_id = k.war_id AND wa.alliance_id = %[1]s)))
          OR (%[2]s IS NOT NULL AND (
                  %[2]s = w.defender_corporation_id
                  OR EXISTS (SELECT 1 FROM war_allies wa
                             WHERE wa.war_id = k.war_id AND wa.corporation_id = %[2]s)))
            THEN 2::smallint
        ELSE NULL
    END`, allianceCol, corporationCol)
}
