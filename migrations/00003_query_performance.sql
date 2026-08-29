-- Improve the live backfill, global statistics, and matchup access paths.

-- +goose NO TRANSACTION

-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS characters_history_backfill_idx
    ON characters (last_active DESC NULLS LAST, character_id)
    WHERE deleted IS NOT TRUE
      AND corporation_history_fetched_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS killmails_matchup_idx
    ON killmails (victim_ship_type_id, killmail_time, killmail_id)
    WHERE is_solo IS TRUE;

CREATE INDEX CONCURRENTLY IF NOT EXISTS killmail_attackers_matchup_idx
    ON killmail_attackers (ship_type_id, killmail_id)
    WHERE character_id IS NOT NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS killmail_attackers_matchup_idx;
DROP INDEX CONCURRENTLY IF EXISTS killmails_matchup_idx;
DROP INDEX CONCURRENTLY IF EXISTS characters_history_backfill_idx;
