-- +goose NO TRANSACTION
-- +goose Up
-- Match the faction and recent-date predicates in one range. The existing
-- faction/id index still serves cursor paging; keep it for that access path.
CREATE INDEX CONCURRENTLY IF NOT EXISTS killmails_fw_winner_time_idx
    ON killmails (fw_winner_faction_id, killmail_time DESC, killmail_id DESC)
    WHERE fw_winner_faction_id IS NOT NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_index
        WHERE indexrelid = 'killmails_fw_winner_time_idx'::regclass
          AND indisvalid AND indisready
    ) THEN
        RAISE EXCEPTION 'killmails_fw_winner_time_idx is invalid; remove the failed index before retrying';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS killmails_fw_winner_time_idx;
