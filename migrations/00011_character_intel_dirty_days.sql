-- +goose Up
CREATE TABLE IF NOT EXISTS character_intel_dirty_days (
    activity_date date PRIMARY KEY,
    dirtied_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS character_intel_dirty_days;
