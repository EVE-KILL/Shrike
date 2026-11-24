-- +goose Up
ALTER TABLE battles ADD COLUMN IF NOT EXISTS report_pending boolean NOT NULL DEFAULT true;
ALTER TABLE battles ADD COLUMN IF NOT EXISTS detector_fingerprint bytea;

CREATE TABLE IF NOT EXISTS battle_report_snapshots (
    battle_id integer PRIMARY KEY REFERENCES battles(battle_id) ON DELETE CASCADE,
    report jsonb NOT NULL,
    loss_summary jsonb NOT NULL,
    loss_events jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS battle_aliases (
    battle_id integer PRIMARY KEY,
    canonical_battle_id integer NOT NULL REFERENCES battles(battle_id) ON DELETE CASCADE,
    CHECK (battle_id <> canonical_battle_id)
);
CREATE INDEX IF NOT EXISTS battle_aliases_canonical_idx ON battle_aliases (canonical_battle_id);

CREATE TABLE IF NOT EXISTS battle_dirty_days (
    solar_system_id integer NOT NULL,
    activity_date date NOT NULL,
    dirtied_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (solar_system_id, activity_date)
);
CREATE INDEX IF NOT EXISTS battle_dirty_days_queued_idx ON battle_dirty_days (dirtied_at);

-- +goose Down
DROP TABLE battle_dirty_days;
DROP TABLE battle_aliases;
DROP TABLE battle_report_snapshots;
ALTER TABLE battles DROP COLUMN detector_fingerprint;
ALTER TABLE battles DROP COLUMN report_pending;
