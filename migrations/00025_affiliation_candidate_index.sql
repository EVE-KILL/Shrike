-- +goose NO TRANSACTION
-- +goose Up
-- Match affiliation refresh ordering and cover its activity predicate without
-- taking a write-blocking index build on the large characters table.
CREATE INDEX CONCURRENTLY IF NOT EXISTS characters_affiliation_due_idx
    ON characters (updated_at ASC NULLS FIRST, character_id)
    INCLUDE (last_active)
    WHERE deleted IS NOT TRUE AND character_id > 0;

-- A cancelled concurrent build may leave an invalid index. Fail explicitly
-- instead of recording this migration as applied when retrying such a build.
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_index
        WHERE indexrelid = 'characters_affiliation_due_idx'::regclass AND indisvalid
    ) THEN
        RAISE EXCEPTION 'characters_affiliation_due_idx is invalid; remove the failed index before retrying';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS characters_affiliation_due_idx;
