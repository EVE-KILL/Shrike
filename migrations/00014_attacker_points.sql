-- +goose Up
ALTER TABLE killmail_attackers
    ADD COLUMN IF NOT EXISTS points bigint NOT NULL DEFAULT 0;

COMMENT ON COLUMN killmail_attackers.points IS
    'Conserved share of killmails.points allocated to this player attacker; NPC and duplicate character rows receive zero';

-- +goose Down
ALTER TABLE killmail_attackers
    DROP COLUMN IF EXISTS points;
