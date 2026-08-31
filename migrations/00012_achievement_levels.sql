-- +goose Up
ALTER TABLE entity_achievements
    ADD COLUMN IF NOT EXISTS level_thresholds integer[] NOT NULL DEFAULT ARRAY[]::integer[],
    ADD COLUMN IF NOT EXISTS point_unit integer NOT NULL DEFAULT 0;

-- Do not rewrite the existing achievement table here: production contains
-- tens of millions of rows. Live awards populate both fields on conflict and
-- the explicit achievement rebuild converges historical rows afterward.

-- +goose Down
ALTER TABLE entity_achievements
    DROP COLUMN IF EXISTS point_unit,
    DROP COLUMN IF EXISTS level_thresholds;
