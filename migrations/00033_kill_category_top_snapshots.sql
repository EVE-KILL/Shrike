-- +goose Up
CREATE TABLE IF NOT EXISTS kill_category_top_snapshots (
    kind text PRIMARY KEY,
    prepared_at timestamptz NOT NULL,
    boards jsonb NOT NULL
);

-- +goose Down
DROP TABLE kill_category_top_snapshots;
