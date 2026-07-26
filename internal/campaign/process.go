package campaign

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Recomputing a campaign.
//
// Two stages, and the split is what makes it tractable. First collect candidate
// killmail ids into a scratch table using the narrow per-entity indexes, then
// aggregate over that set. Doing it as one query would mean a join across
// several OR'd predicates that no index serves, over the whole killmails table.
//
// The scratch tables are unlogged and keyed by campaign, so two campaigns can
// recompute concurrently and a crash leaves rows that the next run clears.

// ErrPaused means a campaign has been deliberately taken out of processing.
var ErrPaused = errors.New("campaign processing is paused")

// ProcessingTimeout bounds one recompute at the database.
//
// A campaign scoped too broadly would otherwise hold a connection for minutes.
// Failing is the right outcome — it tells the owner their campaign is too
// broad, where a slow success just makes the sweep unreliable for everyone.
const ProcessingTimeout = 10 * time.Minute

// Result reports one recompute.
type Result struct {
	CampaignID string    `json:"campaign_id"`
	Killmails  int64     `json:"killmails"`
	Sides      int       `json:"sides"`
	Through    time.Time `json:"processed_through"`
}

// SideTotals is one side's aggregate.
type SideTotals struct {
	SideIndex    int16   `json:"side_index"`
	Kills        int64   `json:"kills"`
	Losses       int64   `json:"losses"`
	IskDestroyed float64 `json:"isk_destroyed"`
	IskLost      float64 `json:"isk_lost"`
}

// Process recomputes one campaign's statistics.
func Process(ctx context.Context, pool *pgxpool.Pool, campaignID string) (*Result, error) {
	c, entities, err := Load(ctx, pool, campaignID)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	if c.ProcessingPaused {
		return nil, ErrPaused
	}

	now := time.Now().UTC()
	end := c.EffectiveEnd(now)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if _, err := tx.Exec(ctx,
		`SELECT set_config('statement_timeout', $1, true)`,
		fmt.Sprint(ProcessingTimeout.Milliseconds())); err != nil {
		return nil, err
	}

	// Clear anything a crashed earlier run left behind for this campaign.
	for _, table := range []string{"campaign_scratch_candidates", "campaign_scratch_killmails"} {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf(`DELETE FROM %s WHERE campaign_id = $1`, table), campaignID); err != nil {
			return nil, fmt.Errorf("clear %s: %w", table, err)
		}
	}

	_, err = collectCandidates(ctx, tx, c, entities, end)
	if err != nil {
		return nil, err
	}

	count, err := materializeCandidates(ctx, tx, c, end)
	if err != nil {
		return nil, err
	}
	if err := attributeSides(ctx, tx, c.ID); err != nil {
		return nil, err
	}

	totals, err := aggregateSides(ctx, tx, c.ID)
	if err != nil {
		return nil, err
	}

	if err := storeSideTotals(ctx, tx, campaignID, totals); err != nil {
		return nil, err
	}
	if err := storeEntityTotals(ctx, tx, campaignID); err != nil {
		return nil, err
	}
	if err := storeDailyTotals(ctx, tx, campaignID, IsAreaCampaign(entities, c.Location)); err != nil {
		return nil, err
	}
	if err := storePrizeStandings(ctx, tx, campaignID); err != nil {
		return nil, err
	}

	stats, lastActivity, err := buildStats(ctx, tx, campaignID, IsAreaCampaign(entities, c.Location))
	if err != nil {
		return nil, err
	}

	// processed_through is the gate's memory. Setting it to the effective end
	// rather than to now is deliberate: for a finished campaign they differ,
	// and using now would make the gate believe it had covered a window it had
	// not looked at.
	if _, err := tx.Exec(ctx, `
        UPDATE campaigns SET
            stats = $3::jsonb,
            processed_through = $2,
            last_activity_at = coalesce($4, last_activity_at),
            stats_updated_at = now(),
            status = CASE WHEN status = $5 THEN $6 ELSE status END,
            updated_at = now()
        WHERE campaign_id = $1`,
		campaignID,
		end,
		stats,
		lastActivity,
		StatusPending,
		StatusActive,
	); err != nil {
		return nil, err
	}

	for _, table := range []string{"campaign_scratch_candidates", "campaign_scratch_killmails"} {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf(`DELETE FROM %s WHERE campaign_id = $1`, table), campaignID); err != nil {
			return nil, fmt.Errorf("clean %s: %w", table, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &Result{
		CampaignID: campaignID,
		Killmails:  count,
		Sides:      len(totals),
		Through:    end,
	}, nil
}

// collectCandidates gathers the killmail ids matching the campaign.
//
// One insert per entity leg, each using its own index, rather than one query
// with OR'd predicates that no index can serve. The campaign's validated time
// window is the workload boundary; every matching killmail inside it is kept.
func collectCandidates(ctx context.Context, tx pgx.Tx, c *Campaign, entities []Entity, end time.Time) (int64, error) {
	idLo, okLo, err := boundID(ctx, tx, c.StartTime.Add(-IDBoundSlack), true)
	if err != nil || !okLo {
		return 0, err
	}
	idHi, okHi, err := boundID(ctx, tx, end.Add(IDBoundSlack), false)
	if err != nil || !okHi {
		return 0, err
	}

	chars := entityIDs(entities, EntityCharacter)
	corps := entityIDs(entities, EntityCorporation)
	allies := entityIDs(entities, EntityAlliance)

	legs := []struct {
		column string
		ids    []int32
	}{
		{"victim_alliance_id", allies},
		{"victim_corporation_id", corps},
		{"victim_character_id", chars},
	}
	// An area campaign has no participants, so its location is what selects the
	// killmails rather than merely narrowing them.
	if IsAreaCampaign(entities, c.Location) {
		legs = append(legs,
			struct {
				column string
				ids    []int32
			}{"solar_system_id", c.Location.SystemIDs},
			struct {
				column string
				ids    []int32
			}{"constellation_id", c.Location.ConstellationIDs},
			struct {
				column string
				ids    []int32
			}{"region_id", c.Location.RegionIDs},
		)
	}

	for _, leg := range legs {
		if len(leg.ids) == 0 {
			continue
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
            INSERT INTO campaign_scratch_candidates (campaign_id, killmail_id)
            SELECT $1, killmail_id FROM killmails
            WHERE %s = ANY($2::int[])
              AND killmail_id BETWEEN $3 AND $4
              AND killmail_time >= $5::timestamptz AND killmail_time <= $6::timestamptz
            ON CONFLICT DO NOTHING`, leg.column),
			c.ID, leg.ids, idLo, idHi, c.StartTime, end); err != nil {
			return 0, fmt.Errorf("collect %s candidates: %w", leg.column, err)
		}
	}

	// Attacker legs bring in the kills where a participant was on the winning
	// side rather than the losing one.
	for _, leg := range []struct {
		column string
		ids    []int32
	}{
		{"alliance_id", allies},
		{"corporation_id", corps},
		{"character_id", chars},
	} {
		if len(leg.ids) == 0 {
			continue
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
            INSERT INTO campaign_scratch_candidates (campaign_id, killmail_id)
            SELECT DISTINCT $1, a.killmail_id FROM killmail_attackers a
            WHERE a.%s = ANY($2::int[])
              AND a.killmail_time >= $3::timestamptz AND a.killmail_time <= $4::timestamptz
            ON CONFLICT DO NOTHING`, leg.column),
			c.ID, leg.ids, c.StartTime, end); err != nil {
			return 0, fmt.Errorf("collect attacker %s candidates: %w", leg.column, err)
		}
	}

	var count int64
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM campaign_scratch_candidates WHERE campaign_id = $1`,
		c.ID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// boundID converts a time to a killmail id bound.
func boundID(ctx context.Context, tx pgx.Tx, at time.Time, lower bool) (int64, bool, error) {
	order, cmp := "ASC", ">="
	if !lower {
		order, cmp = "DESC", "<="
	}
	var id int64
	err := tx.QueryRow(ctx, fmt.Sprintf(`
        SELECT killmail_id FROM killmails
        WHERE killmail_time %s $1::timestamptz
        ORDER BY killmail_time %s LIMIT 1`, cmp, order), at).Scan(&id)
	if isNoRows(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// aggregateSides tallies each side over the collected candidates.
//
// A killmail counts as a kill for a side when one of its members was an
// attacker and the victim was not on that same side. It counts as a loss when
// the victim was a member. Same-side fire is therefore a loss, not a kill.
func aggregateSides(ctx context.Context, tx pgx.Tx, campaignID string) ([]SideTotals, error) {
	var out []SideTotals
	rows, err := tx.Query(ctx, `
        SELECT side.side_index,
               count(killmail.killmail_id) FILTER (
                   WHERE ((killmail.attacker_mask >> side.side_index::int) & 1) = 1
                     AND killmail.victim_side IS DISTINCT FROM side.side_index
               ),
               count(killmail.killmail_id) FILTER (
                   WHERE killmail.victim_side = side.side_index
               ),
               coalesce(sum(killmail.adj_value) FILTER (
                   WHERE ((killmail.attacker_mask >> side.side_index::int) & 1) = 1
                     AND killmail.victim_side IS DISTINCT FROM side.side_index
               ), 0),
               coalesce(sum(killmail.adj_value) FILTER (
                   WHERE killmail.victim_side = side.side_index
               ), 0)
        FROM campaign_sides side
        LEFT JOIN campaign_scratch_killmails killmail
          ON killmail.campaign_id = side.campaign_id
        WHERE side.campaign_id = $1
        GROUP BY side.side_index
        ORDER BY side.side_index`,
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t SideTotals
		if err := rows.Scan(
			&t.SideIndex,
			&t.Kills,
			&t.Losses,
			&t.IskDestroyed,
			&t.IskLost,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// storeSideTotals writes the computed aggregate onto each side.
//
// Update rather than upsert. A side is defined when the campaign is created and
// carries a name this job has no business inventing — a missing row means the
// campaign definition is inconsistent, and creating a nameless side would hide
// that rather than surface it.
func storeSideTotals(ctx context.Context, tx pgx.Tx, campaignID string, totals []SideTotals) error {
	for _, t := range totals {
		tag, err := tx.Exec(ctx, `
            UPDATE campaign_sides SET
                kills = $3, losses = $4, isk_destroyed = $5, isk_lost = $6
            WHERE campaign_id = $1 AND side_index = $2`,
			campaignID, t.SideIndex, t.Kills, t.Losses, t.IskDestroyed, t.IskLost)
		if err != nil {
			return fmt.Errorf("store side %d totals: %w", t.SideIndex, err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf(
				"campaign %s has entities on side %d but no such side is defined",
				campaignID, t.SideIndex)
		}
	}
	return nil
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
