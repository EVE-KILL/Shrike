-- +goose Up
CREATE TABLE IF NOT EXISTS killmail_fleet_affiliations (
    killmail_id integer PRIMARY KEY REFERENCES killmails(killmail_id) ON DELETE CASCADE,
    kinds smallint[] NOT NULL,
    entity_ids integer[] NOT NULL,
    sole_character_ids integer[] NOT NULL,
    CHECK (cardinality(kinds) = cardinality(entity_ids)
       AND cardinality(kinds) = cardinality(sole_character_ids))
);

-- +goose Down
DROP TABLE killmail_fleet_affiliations;
