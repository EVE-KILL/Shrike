-- +goose NO TRANSACTION
-- +goose Up
-- Affiliation updates clear visibility bits across this large table. Vacuum
-- after roughly 105k changes instead of 420k so covering scans stay effective.
-- Keep the existing analyze threshold and autovacuum cost limits.
ALTER TABLE characters SET (autovacuum_vacuum_scale_factor = 0.005);

-- Coalition/alliance member totals need only the alliance key for live members.
-- Exclude unaffiliated characters and cover the deleted predicate so counting
-- members can use an index-only scan instead of reading their character rows.
CREATE INDEX CONCURRENTLY IF NOT EXISTS characters_live_alliance_members_idx
    ON characters (alliance_id)
    WHERE alliance_id IS NOT NULL AND deleted IS NOT TRUE;

-- Cover both dates in the active refresh window so PostgreSQL can discard
-- recently checked characters in the index rather than fetching their rows.
-- Keep the existing due-order index for the separate dormant refresh path.
CREATE INDEX CONCURRENTLY IF NOT EXISTS characters_active_affiliation_due_idx
    ON characters (last_active, updated_at ASC NULLS FIRST, character_id)
    WHERE deleted IS NOT TRUE AND character_id > 0 AND last_active IS NOT NULL;

-- A cancelled concurrent build leaves an invalid index behind. Never mark the
-- migration complete until this index is actually usable.
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_index
        WHERE indexrelid = 'characters_live_alliance_members_idx'::regclass
          AND indisvalid AND indisready
    ) THEN
        RAISE EXCEPTION 'characters_live_alliance_members_idx is invalid; remove the failed index before retrying';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_index
        WHERE indexrelid = 'characters_active_affiliation_due_idx'::regclass
          AND indisvalid AND indisready
    ) THEN
        RAISE EXCEPTION 'characters_active_affiliation_due_idx is invalid; remove the failed index before retrying';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS characters_active_affiliation_due_idx;
DROP INDEX CONCURRENTLY IF EXISTS characters_live_alliance_members_idx;
ALTER TABLE characters SET (autovacuum_vacuum_scale_factor = 0.02);
