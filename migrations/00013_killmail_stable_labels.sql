-- +goose Up
-- +goose NO TRANSACTION
ALTER TABLE killmails
    ADD COLUMN IF NOT EXISTS is_awox boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_capital_involved boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_super_involved boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_titan_involved boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_at_ship_involved boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS fw_winner_faction_id integer;

CREATE INDEX CONCURRENTLY IF NOT EXISTS killmails_awox_idx ON killmails (killmail_id DESC) WHERE is_awox;
CREATE INDEX CONCURRENTLY IF NOT EXISTS killmails_capital_involved_idx ON killmails (killmail_id DESC) WHERE is_capital_involved;
CREATE INDEX CONCURRENTLY IF NOT EXISTS killmails_super_involved_idx ON killmails (killmail_id DESC) WHERE is_super_involved;
CREATE INDEX CONCURRENTLY IF NOT EXISTS killmails_titan_involved_idx ON killmails (killmail_id DESC) WHERE is_titan_involved;
CREATE INDEX CONCURRENTLY IF NOT EXISTS killmails_at_ship_involved_idx ON killmails (killmail_id DESC) WHERE is_at_ship_involved;
CREATE INDEX CONCURRENTLY IF NOT EXISTS killmails_fw_winner_idx ON killmails (fw_winner_faction_id, killmail_id DESC)
    WHERE fw_winner_faction_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS killmails_fw_winner_idx;
DROP INDEX IF EXISTS killmails_at_ship_involved_idx;
DROP INDEX IF EXISTS killmails_titan_involved_idx;
DROP INDEX IF EXISTS killmails_super_involved_idx;
DROP INDEX IF EXISTS killmails_capital_involved_idx;
DROP INDEX IF EXISTS killmails_awox_idx;

ALTER TABLE killmails
    DROP COLUMN IF EXISTS fw_winner_faction_id,
    DROP COLUMN IF EXISTS is_at_ship_involved,
    DROP COLUMN IF EXISTS is_titan_involved,
    DROP COLUMN IF EXISTS is_super_involved,
    DROP COLUMN IF EXISTS is_capital_involved,
    DROP COLUMN IF EXISTS is_awox;
