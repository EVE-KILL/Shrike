-- +goose NO TRANSACTION

-- +goose Up

CREATE INDEX CONCURRENTLY IF NOT EXISTS characters_affiliation_due_idx
    ON characters (updated_at ASC NULLS FIRST, character_id)
    INCLUDE (last_active)
    WHERE deleted IS NOT TRUE AND character_id > 0;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_km_total_value_covering
    ON killmails (total_value DESC, killmail_id DESC)
    INCLUDE (killmail_time);

DROP INDEX CONCURRENTLY IF EXISTS idx_km_total_value;

ALTER INDEX idx_km_total_value_covering RENAME TO idx_km_total_value;

-- +goose Down

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_km_total_value_legacy
    ON killmails (total_value DESC, killmail_id DESC);

DROP INDEX CONCURRENTLY IF EXISTS idx_km_total_value;

ALTER INDEX idx_km_total_value_legacy RENAME TO idx_km_total_value;

DROP INDEX CONCURRENTLY IF EXISTS characters_affiliation_due_idx;
