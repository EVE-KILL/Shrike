// Package maintenance holds the housekeeping jobs — the ones that are a SQL
// statement and a retention policy rather than an integration.
//
// They are collected here rather than spread across the workers because they
// have nothing in common with the work they clean up after, and everything in
// common with each other: each is bounded, idempotent, and safe to run twice.
package maintenance

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PurgeFeed removes feed rows older than a year.
func PurgeFeed(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tag, err := pool.Exec(ctx,
		`DELETE FROM feed_queue WHERE inserted_at < CURRENT_DATE - 365`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// FittingsRetentionDays bounds the fittings tables to the window the item
// pages query over.
const FittingsRetentionDays = 90

// FittingsPurgeResult reports what each phase removed.
type FittingsPurgeResult struct {
	Linkages int64 `json:"linkages"`
	Fits     int64 `json:"fits"`
	Items    int64 `json:"items"`
}

// PurgeFittings trims the fittings tables.
//
// The three phases must run in this order. Expiring the killmail linkages is
// what orphans a fit identity, and dropping the orphaned identity is what
// orphans its items — so running them out of order simply finds nothing to do
// and leaves the largest table untouched, which is where the space actually is.
//
// Re-insertion is cheap, so there is no attempt to keep a fit around in case
// somebody flies it again. The retention window is the authoritative boundary.
func PurgeFittings(ctx context.Context, pool *pgxpool.Pool) (FittingsPurgeResult, error) {
	var out FittingsPurgeResult
	cutoff := fmt.Sprintf("%d days", FittingsRetentionDays)

	tag, err := pool.Exec(ctx,
		`DELETE FROM killmail_fittings WHERE kill_time < now() - $1::interval`, cutoff)
	if err != nil {
		return out, fmt.Errorf("expire killmail linkages: %w", err)
	}
	out.Linkages = tag.RowsAffected()

	tag, err = pool.Exec(ctx, `
        DELETE FROM fittings f
        WHERE NOT EXISTS (SELECT 1 FROM killmail_fittings kf WHERE kf.fit_hash = f.fit_hash)`)
	if err != nil {
		return out, fmt.Errorf("sweep orphaned fits: %w", err)
	}
	out.Fits = tag.RowsAffected()

	tag, err = pool.Exec(ctx, `
        DELETE FROM fitting_items fi
        WHERE NOT EXISTS (SELECT 1 FROM fittings f WHERE f.fit_hash = fi.fit_hash)`)
	if err != nil {
		return out, fmt.Errorf("sweep orphaned fit items: %w", err)
	}
	out.Items = tag.RowsAffected()

	return out, nil
}

// PriceCompactionAfterDays is how much daily price history is kept before it
// is folded into weekly averages.
const PriceCompactionAfterDays = 90

// MinDaysBeforeCompaction guards against compacting a database that has barely
// any history. Without it, a fresh install with a week of prices would fold
// that week into a single point and lose it.
const MinDaysBeforeCompaction = 180

// CompactPrices folds daily price rows older than the retention window into
// weekly averages.
//
// The weekly row is written at the Monday of its week, which is also a date the
// daily rows could occupy — so the delete afterwards deliberately spares any
// row already sitting on a week boundary. Deleting it would remove the
// aggregate that was just written.
func CompactPrices(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var days int
	if err := pool.QueryRow(ctx, `
        SELECT count(DISTINCT date) FROM prices
        WHERE date >= (current_date - interval '180 days')`).Scan(&days); err != nil {
		return 0, err
	}
	if days < MinDaysBeforeCompaction {
		return 0, nil
	}

	before := time.Now().UTC().AddDate(0, 0, -PriceCompactionAfterDays).Format("2006-01-02")

	if _, err := pool.Exec(ctx, `
        WITH weekly AS (
            SELECT type_id, region_id,
                   date_trunc('week', date::timestamp)::date AS week_date,
                   avg(average) AS average,
                   max(highest) AS highest,
                   min(lowest) AS lowest,
                   sum(order_count)::int AS order_count,
                   sum(volume)::bigint AS volume
            FROM prices
            WHERE date < $1::date
            GROUP BY type_id, region_id, date_trunc('week', date::timestamp)::date
            HAVING count(*) > 1
        )
        INSERT INTO prices (type_id, region_id, date, average, highest, lowest, order_count, volume)
        SELECT type_id, region_id, week_date, average, highest, lowest, order_count, volume
        FROM weekly
        ON CONFLICT (type_id, region_id, date) DO UPDATE SET
            average = EXCLUDED.average,
            highest = EXCLUDED.highest,
            lowest = EXCLUDED.lowest,
            order_count = EXCLUDED.order_count,
            volume = EXCLUDED.volume`, before); err != nil {
		return 0, fmt.Errorf("write weekly aggregates: %w", err)
	}

	tag, err := pool.Exec(ctx, `
        DELETE FROM prices
        WHERE date < $1::date
          AND date != date_trunc('week', date::timestamp)::date`, before)
	if err != nil {
		return 0, fmt.Errorf("drop compacted daily rows: %w", err)
	}
	return tag.RowsAffected(), nil
}

// SnapshotEntities records today's member statistics per corporation and
// alliance.
//
// Keyed by date and upserted, so running twice in a day overwrites rather than
// duplicating — which matters because the numbers move during the day and the
// last run of the day is the one worth keeping.
func SnapshotEntities(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var total int64
	today := time.Now().UTC().Format("2006-01-02")

	// The two statements are identical apart from the grouping column, so the
	// shape is written once and the column substituted. The values are from
	// this fixed pair, never from input.
	for _, e := range []struct{ kind, column string }{
		{"corporation", "corporation_id"},
		{"alliance", "alliance_id"},
	} {
		tag, err := pool.Exec(ctx, fmt.Sprintf(`
            INSERT INTO entity_snapshots (
                entity_type, entity_id, date, member_count, avg_sec_status,
                pirate_members, carebear_members, neutral_members
            )
            SELECT $1, c.%[1]s, $2::date,
                   count(*)::int,
                   avg(c.security_status)::real,
                   count(*) FILTER (WHERE c.security_status < -1.5)::int,
                   count(*) FILTER (WHERE c.security_status > 1.5)::int,
                   count(*) FILTER (WHERE c.security_status BETWEEN -1.5 AND 1.5)::int
            FROM characters c
            WHERE c.%[1]s IS NOT NULL AND c.deleted IS NOT TRUE
            GROUP BY c.%[1]s
            ON CONFLICT (entity_type, entity_id, date) DO UPDATE SET
                member_count = EXCLUDED.member_count,
                avg_sec_status = EXCLUDED.avg_sec_status,
                pirate_members = EXCLUDED.pirate_members,
                carebear_members = EXCLUDED.carebear_members,
                neutral_members = EXCLUDED.neutral_members`, e.column), e.kind, today)
		if err != nil {
			return total, fmt.Errorf("snapshot %ss: %w", e.kind, err)
		}
		total += tag.RowsAffected()

		// Achievement totals are character-scoped, then rolled up through the
		// character's current corporation/alliance. This is a second pass
		// because entity_snapshots has to exist before it can be updated.
		if _, err := pool.Exec(ctx, fmt.Sprintf(`
            WITH character_points AS (
                SELECT entity_id AS character_id, sum(points)::int AS total_points
                FROM entity_achievements
                GROUP BY entity_id
                HAVING sum(points) > 0
            ),
            entity_totals AS (
                SELECT c.%[1]s AS entity_id,
                       sum(cp.total_points)::int AS total_points,
                       avg(cp.total_points)::real AS avg_points
                FROM character_points cp
                JOIN characters c ON c.character_id = cp.character_id
                WHERE c.%[1]s IS NOT NULL AND c.deleted IS NOT TRUE
                GROUP BY c.%[1]s
            ),
            entity_top AS (
                SELECT DISTINCT ON (c.%[1]s)
                       c.%[1]s AS entity_id,
                       cp.character_id AS top_id,
                       cp.total_points AS top_points
                FROM character_points cp
                JOIN characters c ON c.character_id = cp.character_id
                WHERE c.%[1]s IS NOT NULL AND c.deleted IS NOT TRUE
                ORDER BY c.%[1]s, cp.total_points DESC, cp.character_id
            )
            UPDATE entity_snapshots snapshot SET
                total_achievement_points = aggregate.total_points,
                avg_achievement_points = aggregate.avg_points,
                top_achiever_id = top.top_id,
                top_achiever_points = top.top_points
            FROM entity_totals aggregate
            LEFT JOIN entity_top top ON top.entity_id = aggregate.entity_id
            WHERE snapshot.entity_type = $1
              AND snapshot.entity_id = aggregate.entity_id
              AND snapshot.date = $2::date`, e.column), e.kind, today); err != nil {
			return total, fmt.Errorf("snapshot %s achievements: %w", e.kind, err)
		}
	}
	return total, nil
}

// ReconcileDays is how far back the daily kill counts are rebuilt.
//
// The rollup is bumped as each killmail is processed, and that bump logs and
// continues rather than failing the killmail — so drift is possible, rare, and
// worth correcting before it reaches a page anybody looks at. A week is far
// more than enough.
const ReconcileDays = 7

// ReconcileDailyCounts rebuilds the recent daily kill counts from the killmails
// themselves, correcting any drift in the incremental rollup.
func ReconcileDailyCounts(ctx context.Context, pool *pgxpool.Pool, predicates map[string]string) (int64, error) {
	from := time.Now().UTC().AddDate(0, 0, -ReconcileDays).Format("2006-01-02")
	// Exclusive upper bound of tomorrow, so today is included.
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	var total int64
	for kind, predicate := range predicates {
		tag, err := pool.Exec(ctx, fmt.Sprintf(`
            INSERT INTO kills_daily_count (date, type, count)
            SELECT (k.killmail_time AT TIME ZONE 'UTC')::date, $1, count(*)
            FROM killmails k
            WHERE k.killmail_time >= $2::date
              AND k.killmail_time <  $3::date
              AND %s
            GROUP BY (k.killmail_time AT TIME ZONE 'UTC')::date
            ON CONFLICT (date, type) DO UPDATE SET count = EXCLUDED.count`, predicate),
			kind, from, to)
		if err != nil {
			return total, fmt.Errorf("reconcile %s: %w", kind, err)
		}
		total += tag.RowsAffected()
	}
	return total, nil
}

// Analyze refreshes planner statistics on the tables large enough for
// autovacuum's proportional thresholds to leave stale.
//
// On a table with hundreds of millions of rows autovacuum waits for tens of
// millions of changes before analysing, by which time the planner has been
// choosing badly for hours. Doing it on a schedule costs less than the
// sequential scans it prevents.
func Analyze(ctx context.Context, pool *pgxpool.Pool, tables []string) (int, error) {
	for i, t := range tables {
		// Identifiers come from the caller's fixed list, never from input.
		if _, err := pool.Exec(ctx, "ANALYZE "+t); err != nil {
			return i, fmt.Errorf("analyze %s: %w", t, err)
		}
	}
	return len(tables), nil
}
