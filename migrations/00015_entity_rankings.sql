-- +goose Up
CREATE TABLE IF NOT EXISTS entity_rankings (
    entity_type smallint NOT NULL,
    entity_id integer NOT NULL,
    ranking_window smallint NOT NULL,
    combat_points bigint NOT NULL DEFAULT 0,
    achievement_points bigint NOT NULL DEFAULT 0,
    eve_kill_rating integer NOT NULL DEFAULT 0,
    combat_rank integer NOT NULL,
    achievement_rank integer,
    overall_rank integer NOT NULL,
    population integer NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_type, entity_id, ranking_window)
);

CREATE INDEX IF NOT EXISTS entity_rankings_explorer_idx
    ON entity_rankings (ranking_window, entity_type, overall_rank);

-- +goose Down
DROP TABLE IF EXISTS entity_rankings;
