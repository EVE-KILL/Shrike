-- +goose Up
CREATE TABLE IF NOT EXISTS roam_reports (
    id text PRIMARY KEY CHECK (id ~ '^[0-9a-f]{32}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    start_time timestamptz NOT NULL,
    end_time timestamptz NOT NULL,
    report jsonb NOT NULL,
    CONSTRAINT roam_reports_window_check CHECK (start_time < end_time)
);

-- +goose Down
DROP TABLE roam_reports;
