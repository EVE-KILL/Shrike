-- +goose Up
ALTER TABLE fitting_stats
    ADD COLUMN IF NOT EXISTS shield_hp double precision,
    ADD COLUMN IF NOT EXISTS armor_hp double precision,
    ADD COLUMN IF NOT EXISTS hull_hp double precision,
    ADD COLUMN IF NOT EXISTS shield_em_resist double precision,
    ADD COLUMN IF NOT EXISTS shield_thermal_resist double precision,
    ADD COLUMN IF NOT EXISTS shield_kinetic_resist double precision,
    ADD COLUMN IF NOT EXISTS shield_explosive_resist double precision,
    ADD COLUMN IF NOT EXISTS armor_em_resist double precision,
    ADD COLUMN IF NOT EXISTS armor_thermal_resist double precision,
    ADD COLUMN IF NOT EXISTS armor_kinetic_resist double precision,
    ADD COLUMN IF NOT EXISTS armor_explosive_resist double precision,
    ADD COLUMN IF NOT EXISTS hull_em_resist double precision,
    ADD COLUMN IF NOT EXISTS hull_thermal_resist double precision,
    ADD COLUMN IF NOT EXISTS hull_kinetic_resist double precision,
    ADD COLUMN IF NOT EXISTS hull_explosive_resist double precision;

-- +goose Down
ALTER TABLE fitting_stats
    DROP COLUMN IF EXISTS hull_explosive_resist,
    DROP COLUMN IF EXISTS hull_kinetic_resist,
    DROP COLUMN IF EXISTS hull_thermal_resist,
    DROP COLUMN IF EXISTS hull_em_resist,
    DROP COLUMN IF EXISTS armor_explosive_resist,
    DROP COLUMN IF EXISTS armor_kinetic_resist,
    DROP COLUMN IF EXISTS armor_thermal_resist,
    DROP COLUMN IF EXISTS armor_em_resist,
    DROP COLUMN IF EXISTS shield_explosive_resist,
    DROP COLUMN IF EXISTS shield_kinetic_resist,
    DROP COLUMN IF EXISTS shield_thermal_resist,
    DROP COLUMN IF EXISTS shield_em_resist,
    DROP COLUMN IF EXISTS hull_hp,
    DROP COLUMN IF EXISTS armor_hp,
    DROP COLUMN IF EXISTS shield_hp;
