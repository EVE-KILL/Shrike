-- Daily character-intelligence facts. Keeping the date in the key lets every
-- consumer build its own rolling window and lets reconciliation replace one
-- bounded day without disturbing readers of the other days.

-- +goose Up
CREATE TABLE character_intel_daily (
    character_id integer NOT NULL,
    activity_date date NOT NULL,
    appearances integer NOT NULL DEFAULT 0,
    solo integer NOT NULL DEFAULT 0,
    small_gang integer NOT NULL DEFAULT 0,
    mid_gang integer NOT NULL DEFAULT 0,
    fleet integer NOT NULL DEFAULT 0,
    blob integer NOT NULL DEFAULT 0,
    sum_attacker_count bigint NOT NULL DEFAULT 0,
    monitor_appearances integer NOT NULL DEFAULT 0,
    zero_damage_fleet integer NOT NULL DEFAULT 0,
    highsec integer NOT NULL DEFAULT 0,
    lowsec integer NOT NULL DEFAULT 0,
    nullsec integer NOT NULL DEFAULT 0,
    gank_appearances integer NOT NULL DEFAULT 0,
    awox_kills integer NOT NULL DEFAULT 0,
    losses integer NOT NULL DEFAULT 0,
    cyno_losses integer NOT NULL DEFAULT 0,
    cheap_deaths integer NOT NULL DEFAULT 0,
    baited_deaths integer NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, activity_date)
);

CREATE INDEX character_intel_daily_date_idx
    ON character_intel_daily (activity_date, character_id);

CREATE TABLE character_intel_ship_daily (
    character_id integer NOT NULL,
    activity_date date NOT NULL,
    ship_type_id integer NOT NULL,
    appearances integer NOT NULL DEFAULT 0,
    losses integer NOT NULL DEFAULT 0,
    last_appearance_at timestamp with time zone,
    last_loss_id integer,
    last_loss_at timestamp with time zone,
    PRIMARY KEY (character_id, activity_date, ship_type_id)
);

CREATE INDEX character_intel_ship_daily_date_idx
    ON character_intel_ship_daily (activity_date, character_id);

CREATE TABLE character_intel_target_daily (
    character_id integer NOT NULL,
    activity_date date NOT NULL,
    alliance_id integer NOT NULL,
    appearances integer NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, activity_date, alliance_id)
);

CREATE INDEX character_intel_target_daily_date_idx
    ON character_intel_target_daily (activity_date, character_id);

-- A date is readable from the rollup only after all three tables were replaced
-- successfully in the same transaction.
CREATE TABLE character_intel_rollup_days (
    activity_date date PRIMARY KEY,
    refreshed_at timestamp with time zone NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE character_intel_rollup_days;
DROP TABLE character_intel_target_daily;
DROP TABLE character_intel_ship_daily;
DROP TABLE character_intel_daily;
