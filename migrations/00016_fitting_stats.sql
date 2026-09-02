-- +goose Up
CREATE TABLE IF NOT EXISTS fitting_stats (
    fit_hash text PRIMARY KEY REFERENCES fittings(fit_hash) ON DELETE CASCADE,
    ship_type_id integer NOT NULL,
    skill_level smallint NOT NULL DEFAULT 5 CHECK (skill_level BETWEEN 0 AND 5),
    dps_with_reload double precision, dps_without_reload double precision, alpha double precision,
    ehp double precision, shield_ehp double precision, armor_ehp double precision, hull_ehp double precision,
    shield_boost double precision, shield_effective_boost double precision,
    armor_repair double precision, armor_effective_repair double precision,
    hull_repair double precision, hull_effective_repair double precision,
    passive_shield double precision, passive_shield_effective double precision,
    remote_shield double precision, remote_armor double precision, remote_hull double precision,
    remote_cap double precision, neut double precision, nos double precision,
    cap_stable boolean NOT NULL DEFAULT false, cap_depletes_in double precision,
    cap_capacity double precision, cap_peak_delta double precision,
    max_velocity double precision, align_time double precision, signature_radius double precision,
    max_target_range double precision, scan_resolution double precision,
    engine_version text NOT NULL, sde_version text NOT NULL,
    calculated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS fitting_stats_ship_dps_idx ON fitting_stats (ship_type_id, dps_with_reload DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS fitting_stats_ship_ehp_idx ON fitting_stats (ship_type_id, ehp DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS fitting_stats_ship_speed_idx ON fitting_stats (ship_type_id, max_velocity DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS fitting_stats_ship_align_idx ON fitting_stats (ship_type_id, align_time ASC NULLS LAST);

-- +goose Down
DROP TABLE IF EXISTS fitting_stats;
