-- +goose Up
CREATE TABLE IF NOT EXISTS fitting_stat_distribution_summaries (
    ship_type_id integer NOT NULL,
    window_days smallint NOT NULL,
    metric text NOT NULL,
    fit_count integer NOT NULL,
    observation_count bigint NOT NULL,
    minimum double precision NOT NULL,
    maximum double precision NOT NULL,
    p10 double precision NOT NULL,
    p25 double precision NOT NULL,
    median double precision NOT NULL,
    p75 double precision NOT NULL,
    p90 double precision NOT NULL,
    lower_bound double precision NOT NULL,
    upper_bound double precision NOT NULL,
    calculated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (ship_type_id, window_days, metric)
);

CREATE TABLE IF NOT EXISTS fitting_stat_distribution_buckets (
    ship_type_id integer NOT NULL,
    window_days smallint NOT NULL,
    metric text NOT NULL,
    bucket smallint NOT NULL,
    lower_bound double precision NOT NULL,
    upper_bound double precision NOT NULL,
    fit_count integer NOT NULL,
    observation_count bigint NOT NULL,
    PRIMARY KEY (ship_type_id, window_days, metric, bucket),
    FOREIGN KEY (ship_type_id, window_days, metric)
        REFERENCES fitting_stat_distribution_summaries (ship_type_id, window_days, metric)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS fitting_stat_distribution_metric_idx
    ON fitting_stat_distribution_summaries (window_days, metric, observation_count DESC);

-- +goose Down
DROP TABLE IF EXISTS fitting_stat_distribution_buckets;
DROP TABLE IF EXISTS fitting_stat_distribution_summaries;
