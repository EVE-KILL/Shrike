-- +goose NO TRANSACTION
-- +goose Up
-- Existing installations can have hundreds of thousands of pending reports.
CREATE INDEX CONCURRENTLY IF NOT EXISTS battles_pending_report_idx
    ON battles (updated_at DESC NULLS LAST, start_time DESC, battle_id DESC)
    WHERE report_pending;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_index
        WHERE indexrelid = 'battles_pending_report_idx'::regclass
          AND indisvalid AND indisready
    ) THEN
        RAISE EXCEPTION 'battles_pending_report_idx is invalid; remove the failed index before retrying';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS battles_pending_report_idx;
