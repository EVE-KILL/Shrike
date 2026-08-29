package campaign

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// materializeCandidates copies the time-bounded candidate set into the denormalized
// scratch table used by every aggregate.
//
// This is where location scoping is applied for both area campaigns and
// participant campaigns, and where the custom-price overlay corrects
// supercapital values that Jita history systematically understates.
func materializeCandidates(
	ctx context.Context,
	tx pgx.Tx,
	c *Campaign,
	end time.Time,
) (int64, error) {
	if _, err := tx.Exec(ctx, `
        CREATE TEMP TABLE _campaign_price_deltas ON COMMIT DROP AS
        WITH latest_custom AS (
            SELECT DISTINCT ON (type_id) type_id, price
            FROM custom_prices
            ORDER BY type_id, date DESC
        ),
        latest_market AS (
            SELECT DISTINCT ON (type_id) type_id, average
            FROM prices
            WHERE region_id = 10000002
              AND type_id IN (SELECT type_id FROM latest_custom)
              AND date <= $1::date
            ORDER BY type_id, date DESC
        )
        SELECT custom.type_id,
               custom.price - coalesce(market.average, 0) AS delta
        FROM latest_custom custom
        LEFT JOIN latest_market market USING (type_id)
        WHERE custom.price - coalesce(market.average, 0) <> 0`,
		end.Format("2006-01-02"),
	); err != nil {
		return 0, fmt.Errorf("build campaign price overlay: %w", err)
	}

	if _, err := tx.Exec(ctx, `
        INSERT INTO campaign_scratch_killmails (
            campaign_id, killmail_id, killmail_time, day, adj_value,
            victim_ship_type_id, victim_ship_group_id,
            victim_character_id, victim_corporation_id, victim_alliance_id,
            solar_system_id, region_id
        )
        SELECT
            $1,
            killmail.killmail_id,
            killmail.killmail_time,
            (killmail.killmail_time AT TIME ZONE 'UTC')::date,
            coalesce(killmail.total_value, 0) + coalesce(price.delta, 0),
            killmail.victim_ship_type_id,
            killmail.victim_ship_group_id,
            killmail.victim_character_id,
            killmail.victim_corporation_id,
            killmail.victim_alliance_id,
            killmail.solar_system_id,
            killmail.region_id
        FROM campaign_scratch_candidates candidate
        JOIN killmails killmail
          ON killmail.killmail_id = candidate.killmail_id
        LEFT JOIN _campaign_price_deltas price
          ON price.type_id = killmail.victim_ship_type_id
        WHERE candidate.campaign_id = $1
          AND (
              cardinality($2::int[]) + cardinality($3::int[]) + cardinality($4::int[]) = 0
              OR killmail.solar_system_id = ANY($2::int[])
              OR killmail.constellation_id = ANY($3::int[])
              OR killmail.region_id = ANY($4::int[])
          )
        ON CONFLICT (campaign_id, killmail_id) DO UPDATE SET
            killmail_time = EXCLUDED.killmail_time,
            day = EXCLUDED.day,
            adj_value = EXCLUDED.adj_value,
            victim_ship_type_id = EXCLUDED.victim_ship_type_id,
            victim_ship_group_id = EXCLUDED.victim_ship_group_id,
            victim_character_id = EXCLUDED.victim_character_id,
            victim_corporation_id = EXCLUDED.victim_corporation_id,
            victim_alliance_id = EXCLUDED.victim_alliance_id,
            solar_system_id = EXCLUDED.solar_system_id,
            region_id = EXCLUDED.region_id,
            victim_side = NULL,
            attacker_mask = 0`,
		c.ID,
		c.Location.SystemIDs,
		c.Location.ConstellationIDs,
		c.Location.RegionIDs,
	); err != nil {
		return 0, fmt.Errorf("materialize campaign killmails: %w", err)
	}

	// The joins below switch plans based on the scratch cardinality. This table
	// is shared and unlogged, so refresh its statistics after this campaign's
	// rows have landed.
	if _, err := tx.Exec(ctx, `ANALYZE campaign_scratch_killmails`); err != nil {
		return 0, fmt.Errorf("analyze campaign scratch: %w", err)
	}

	var count int64
	if err := tx.QueryRow(ctx, `
        SELECT count(*)
        FROM campaign_scratch_killmails
        WHERE campaign_id = $1`,
		c.ID,
	).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// attributeSides gives every scratch kill one victim side and an attacker-side
// bitmask. The victim passes intentionally run alliance, corporation, then
// character so the most specific configured entity wins.
func attributeSides(ctx context.Context, tx pgx.Tx, campaignID string) error {
	for _, entityType := range []int16{
		EntityAlliance,
		EntityCorporation,
		EntityCharacter,
	} {
		column := map[int16]string{
			EntityAlliance:    "victim_alliance_id",
			EntityCorporation: "victim_corporation_id",
			EntityCharacter:   "victim_character_id",
		}[entityType]
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
            UPDATE campaign_scratch_killmails killmail
            SET victim_side = entity.side_index
            FROM campaign_side_entities entity
            WHERE killmail.campaign_id = $1
              AND entity.campaign_id = $1
              AND entity.entity_type = $2
              AND killmail.%s = entity.entity_id`,
			column),
			campaignID,
			entityType,
		); err != nil {
			return fmt.Errorf("attribute campaign victim %s: %w", column, err)
		}
	}

	if _, err := tx.Exec(ctx, `
        UPDATE campaign_scratch_killmails killmail
        SET attacker_mask = matched.mask
        FROM (
            SELECT participant.killmail_id,
                   bit_or(1 << participant.side_index::int)::int AS mask
            FROM (
                SELECT attacker.killmail_id, entity.side_index
                FROM campaign_scratch_killmails scratch
                JOIN killmail_attackers attacker
                  ON attacker.killmail_id = scratch.killmail_id
                JOIN campaign_side_entities entity
                  ON entity.campaign_id = $1
                 AND entity.entity_type = $2
                 AND entity.entity_id = attacker.character_id
                WHERE scratch.campaign_id = $1

                UNION ALL

                SELECT attacker.killmail_id, entity.side_index
                FROM campaign_scratch_killmails scratch
                JOIN killmail_attackers attacker
                  ON attacker.killmail_id = scratch.killmail_id
                JOIN campaign_side_entities entity
                  ON entity.campaign_id = $1
                 AND entity.entity_type = $3
                 AND entity.entity_id = attacker.corporation_id
                WHERE scratch.campaign_id = $1

                UNION ALL

                SELECT attacker.killmail_id, entity.side_index
                FROM campaign_scratch_killmails scratch
                JOIN killmail_attackers attacker
                  ON attacker.killmail_id = scratch.killmail_id
                JOIN campaign_side_entities entity
                  ON entity.campaign_id = $1
                 AND entity.entity_type = $4
                 AND entity.entity_id = attacker.alliance_id
                WHERE scratch.campaign_id = $1
            ) participant
            GROUP BY participant.killmail_id
        ) matched
        WHERE killmail.campaign_id = $1
          AND killmail.killmail_id = matched.killmail_id`,
		campaignID,
		EntityCharacter,
		EntityCorporation,
		EntityAlliance,
	); err != nil {
		return fmt.Errorf("attribute campaign attackers: %w", err)
	}
	return nil
}

// restrictToContestedKills drops every scratch killmail that does not cross
// sides — victim on one side, some attacker on a different one.
//
// A campaign with two or more populated sides is a scoreboard BETWEEN them.
// Kills on third parties, losses to third parties and same-side friendly fire
// are participant activity, not part of the matchup, and leaving them in means
// side A's isk_destroyed and side B's isk_lost sum over different populations
// of killmails, so the two can never agree. Filtering the set once here rather
// than per aggregate keeps every downstream number — side totals, per-entity
// rows, the daily series, top lists, ship classes, intel and prize scoring —
// consistently head-to-head off one definition.
//
// Callers skip this for one-sided campaigns (activity trackers with no
// opponent to cross to) and area campaigns (no sides at all). Returns the
// surviving row count.
func restrictToContestedKills(ctx context.Context, tx pgx.Tx, campaignID string) (int64, error) {
	if _, err := tx.Exec(ctx, `
        DELETE FROM campaign_scratch_killmails killmail
        WHERE killmail.campaign_id = $1
          AND (
                killmail.victim_side IS NULL
             OR (killmail.attacker_mask & ~(1 << killmail.victim_side::int)) = 0
          )`,
		campaignID,
	); err != nil {
		return 0, fmt.Errorf("restrict campaign to contested kills: %w", err)
	}

	// Cardinality just dropped hard — replan the aggregate joins.
	if _, err := tx.Exec(ctx, `ANALYZE campaign_scratch_killmails`); err != nil {
		return 0, fmt.Errorf("analyze campaign scratch: %w", err)
	}

	var count int64
	if err := tx.QueryRow(ctx, `
        SELECT count(*)
        FROM campaign_scratch_killmails
        WHERE campaign_id = $1`,
		campaignID,
	).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func storeEntityTotals(ctx context.Context, tx pgx.Tx, campaignID string) error {
	// Reset first so an entity whose previous matches disappeared does not keep
	// stale non-zero totals.
	if _, err := tx.Exec(ctx, `
        UPDATE campaign_side_entities
        SET kills = 0, losses = 0, isk_destroyed = 0, isk_lost = 0
        WHERE campaign_id = $1`,
		campaignID,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
        WITH matched AS (
            SELECT entity.id, killmail.killmail_id, killmail.adj_value
            FROM campaign_side_entities entity
            JOIN campaign_scratch_killmails killmail
              ON killmail.campaign_id = entity.campaign_id
             AND killmail.victim_character_id = entity.entity_id
            WHERE entity.campaign_id = $1 AND entity.entity_type = $2

            UNION ALL

            SELECT entity.id, killmail.killmail_id, killmail.adj_value
            FROM campaign_side_entities entity
            JOIN campaign_scratch_killmails killmail
              ON killmail.campaign_id = entity.campaign_id
             AND killmail.victim_corporation_id = entity.entity_id
            WHERE entity.campaign_id = $1 AND entity.entity_type = $3

            UNION ALL

            SELECT entity.id, killmail.killmail_id, killmail.adj_value
            FROM campaign_side_entities entity
            JOIN campaign_scratch_killmails killmail
              ON killmail.campaign_id = entity.campaign_id
             AND killmail.victim_alliance_id = entity.entity_id
            WHERE entity.campaign_id = $1 AND entity.entity_type = $4
        ),
        totals AS (
            SELECT id, count(killmail_id)::int AS losses,
                   coalesce(sum(adj_value), 0) AS isk_lost
            FROM matched
            GROUP BY id
        )
        UPDATE campaign_side_entities entity
        SET losses = totals.losses,
            isk_lost = totals.isk_lost
        FROM totals
        WHERE entity.id = totals.id`,
		campaignID,
		EntityCharacter,
		EntityCorporation,
		EntityAlliance,
	); err != nil {
		return fmt.Errorf("store campaign entity losses: %w", err)
	}

	if _, err := tx.Exec(ctx, `
        WITH matched AS (
            SELECT entity.id, entity.side_index,
                   scratch.killmail_id, scratch.adj_value, scratch.victim_side
            FROM campaign_scratch_killmails scratch
            JOIN killmail_attackers attacker
              ON attacker.killmail_id = scratch.killmail_id
            JOIN campaign_side_entities entity
              ON entity.campaign_id = $1
             AND entity.entity_type = $2
             AND entity.entity_id = attacker.character_id
            WHERE scratch.campaign_id = $1

            UNION ALL

            SELECT entity.id, entity.side_index,
                   scratch.killmail_id, scratch.adj_value, scratch.victim_side
            FROM campaign_scratch_killmails scratch
            JOIN killmail_attackers attacker
              ON attacker.killmail_id = scratch.killmail_id
            JOIN campaign_side_entities entity
              ON entity.campaign_id = $1
             AND entity.entity_type = $3
             AND entity.entity_id = attacker.corporation_id
            WHERE scratch.campaign_id = $1

            UNION ALL

            SELECT entity.id, entity.side_index,
                   scratch.killmail_id, scratch.adj_value, scratch.victim_side
            FROM campaign_scratch_killmails scratch
            JOIN killmail_attackers attacker
              ON attacker.killmail_id = scratch.killmail_id
            JOIN campaign_side_entities entity
              ON entity.campaign_id = $1
             AND entity.entity_type = $4
             AND entity.entity_id = attacker.alliance_id
            WHERE scratch.campaign_id = $1
        ),
        distinct_kills AS (
            SELECT DISTINCT id, killmail_id, adj_value
            FROM matched
            WHERE victim_side IS DISTINCT FROM side_index
        ),
        totals AS (
            SELECT id, count(*)::int AS kills,
                   coalesce(sum(adj_value), 0) AS isk_destroyed
            FROM distinct_kills
            GROUP BY id
        )
        UPDATE campaign_side_entities entity
        SET kills = totals.kills,
            isk_destroyed = totals.isk_destroyed
        FROM totals
        WHERE entity.id = totals.id`,
		campaignID,
		EntityCharacter,
		EntityCorporation,
		EntityAlliance,
	); err != nil {
		return fmt.Errorf("store campaign entity kills: %w", err)
	}
	return nil
}

func storeDailyTotals(
	ctx context.Context,
	tx pgx.Tx,
	campaignID string,
	area bool,
) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM campaign_stats_daily WHERE campaign_id = $1`,
		campaignID,
	); err != nil {
		return err
	}

	if area {
		_, err := tx.Exec(ctx, `
            INSERT INTO campaign_stats_daily (
                campaign_id, side_index, day,
                kills, losses, isk_destroyed, isk_lost)
            SELECT $1, -1, day,
                   count(*)::int, 0,
                   coalesce(sum(adj_value), 0), 0
            FROM campaign_scratch_killmails
            WHERE campaign_id = $1
            GROUP BY day`,
			campaignID,
		)
		return err
	}

	_, err := tx.Exec(ctx, `
        INSERT INTO campaign_stats_daily (
            campaign_id, side_index, day,
            kills, losses, isk_destroyed, isk_lost)
        SELECT $1, side.side_index, killmail.day,
               count(*) FILTER (
                   WHERE ((killmail.attacker_mask >> side.side_index::int) & 1) = 1
                     AND killmail.victim_side IS DISTINCT FROM side.side_index
               )::int,
               count(*) FILTER (
                   WHERE killmail.victim_side = side.side_index
               )::int,
               coalesce(sum(killmail.adj_value) FILTER (
                   WHERE ((killmail.attacker_mask >> side.side_index::int) & 1) = 1
                     AND killmail.victim_side IS DISTINCT FROM side.side_index
               ), 0),
               coalesce(sum(killmail.adj_value) FILTER (
                   WHERE killmail.victim_side = side.side_index
               ), 0)
        FROM campaign_sides side
        JOIN campaign_scratch_killmails killmail
          ON killmail.campaign_id = side.campaign_id
        WHERE side.campaign_id = $1
        GROUP BY side.side_index, killmail.day`,
		campaignID,
	)
	return err
}

// storePrizeStandings rewrites the live preview only while funding is open.
// Once Finalize changes the pool status, these rows become the immutable payout
// snapshot and normal campaign refreshes leave them alone.
func storePrizeStandings(ctx context.Context, tx pgx.Tx, campaignID string) error {
	var open bool
	err := tx.QueryRow(ctx, `
        SELECT true
        FROM campaign_prize_pools
        WHERE campaign_id = $1 AND status = $2`,
		campaignID,
		PrizePoolFunding,
	).Scan(&open)
	if err != nil {
		if isNoRows(err) {
			return nil
		}
		return err
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM campaign_prize_results WHERE campaign_id = $1`,
		campaignID,
	); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
        WITH pool AS (
            SELECT metric, winner_count, payout_percentages
            FROM campaign_prize_pools
            WHERE campaign_id = $1 AND status = $2
        ),
        campaign_mode AS (
            SELECT EXISTS (
                SELECT 1 FROM campaign_sides WHERE campaign_id = $1
            ) AS has_sides
        ),
        eligible_kills AS (
            SELECT DISTINCT attacker.character_id,
                            scratch.killmail_id,
                            scratch.adj_value
            FROM campaign_scratch_killmails scratch
            JOIN killmail_attackers attacker
              ON attacker.killmail_id = scratch.killmail_id
            CROSS JOIN campaign_mode mode
            WHERE scratch.campaign_id = $1
              AND attacker.character_id IS NOT NULL
              AND (
                  NOT mode.has_sides
                  OR EXISTS (
                      SELECT 1
                      FROM campaign_side_entities side
                      WHERE side.campaign_id = $1
                        AND scratch.victim_side IS DISTINCT FROM side.side_index
                        AND (
                               (side.entity_type = $3 AND side.entity_id = attacker.character_id)
                            OR (side.entity_type = $4 AND side.entity_id = attacker.corporation_id)
                            OR (side.entity_type = $5 AND side.entity_id = attacker.alliance_id)
                        )
                  )
              )
        ),
        killer_scores AS (
            SELECT character_id,
                   count(*)::numeric AS kills,
                   coalesce(sum(adj_value), 0)::numeric AS isk_destroyed
            FROM eligible_kills
            GROUP BY character_id
        ),
        victim_scores AS (
            SELECT scratch.victim_character_id AS character_id,
                   count(*)::numeric AS losses,
                   coalesce(sum(scratch.adj_value), 0)::numeric AS isk_lost
            FROM campaign_scratch_killmails scratch
            CROSS JOIN campaign_mode mode
            WHERE scratch.campaign_id = $1
              AND scratch.victim_character_id IS NOT NULL
              AND (NOT mode.has_sides OR scratch.victim_side IS NOT NULL)
            GROUP BY scratch.victim_character_id
        ),
        scores AS (
            SELECT coalesce(killer.character_id, victim.character_id) AS character_id,
                   coalesce(killer.kills, 0) AS kills,
                   coalesce(killer.isk_destroyed, 0) AS isk_destroyed,
                   coalesce(victim.losses, 0) AS losses,
                   coalesce(victim.isk_lost, 0) AS isk_lost
            FROM killer_scores killer
            FULL OUTER JOIN victim_scores victim
              ON victim.character_id = killer.character_id
        ),
        ranked AS (
            SELECT scores.character_id,
                   CASE pool.metric
                       WHEN 0 THEN scores.kills
                       WHEN 1 THEN scores.losses
                       WHEN 2 THEN scores.isk_destroyed
                       ELSE scores.isk_lost
                   END AS metric_value,
                   CASE pool.metric
                       WHEN 0 THEN scores.isk_destroyed
                       WHEN 1 THEN scores.isk_lost
                       WHEN 2 THEN scores.kills
                       ELSE scores.losses
                   END AS secondary_value,
                   row_number() OVER (
                       ORDER BY
                           CASE pool.metric
                               WHEN 0 THEN scores.kills
                               WHEN 1 THEN scores.losses
                               WHEN 2 THEN scores.isk_destroyed
                               ELSE scores.isk_lost
                           END DESC,
                           CASE pool.metric
                               WHEN 0 THEN scores.isk_destroyed
                               WHEN 1 THEN scores.isk_lost
                               WHEN 2 THEN scores.kills
                               ELSE scores.losses
                           END DESC,
                           scores.character_id
                   ) AS rank,
                   pool.winner_count,
                   pool.payout_percentages
            FROM scores
            CROSS JOIN pool
            WHERE CASE pool.metric
                WHEN 0 THEN scores.kills
                WHEN 1 THEN scores.losses
                WHEN 2 THEN scores.isk_destroyed
                ELSE scores.isk_lost
            END > 0
        )
        INSERT INTO campaign_prize_results (
            campaign_id, rank, character_id, character_name,
            metric_value, secondary_value, payout_percentage)
        SELECT $1,
               ranked.rank,
               ranked.character_id,
               character.name,
               ranked.metric_value,
               ranked.secondary_value,
               ranked.payout_percentages[ranked.rank::int]
        FROM ranked
        LEFT JOIN characters character
          ON character.character_id = ranked.character_id
        WHERE ranked.rank <= ranked.winner_count
        ORDER BY ranked.rank`,
		campaignID,
		PrizePoolFunding,
		EntityCharacter,
		EntityCorporation,
		EntityAlliance,
	)
	return err
}
